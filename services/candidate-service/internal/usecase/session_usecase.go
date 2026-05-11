package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"github.com/tamirat-dejene/veritas/shared/pkg/storage"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
)

type sessionUseCase struct {
	pool           *pgxpool.Pool
	sessionRepo    domain.SessionRepository
	enrollmentRepo domain.EnrollmentRepository
	candidateRepo  domain.CandidateRepository
	examClient     client.ExamServiceClient
	tokenService   domain.EnrollmentTokenService
	publisher      messaging.Publisher
	storage        storage.FileStorage
}

func NewSessionUseCase(
	pool *pgxpool.Pool,
	sRepo domain.SessionRepository,
	eRepo domain.EnrollmentRepository,
	candidateRepo domain.CandidateRepository,
	eClient client.ExamServiceClient,
	tokenService domain.EnrollmentTokenService,
	publisher messaging.Publisher,
	storage storage.FileStorage,
) domain.SessionUseCase {
	return &sessionUseCase{
		pool:           pool,
		sessionRepo:    sRepo,
		enrollmentRepo: eRepo,
		candidateRepo:  candidateRepo,
		examClient:     eClient,
		tokenService:   tokenService,
		publisher:      publisher,
		storage:        storage,
	}
}

func (uc *sessionUseCase) ValidateAccessToken(ctx context.Context, enrollmentID, enterpriseID uuid.UUID) (*domain.ValidateAccessTokenResponse, error) {
	e, err := uc.enrollmentRepo.GetByID(ctx, enrollmentID, enterpriseID)
	if err != nil {
		return nil, err
	}

	return &domain.ValidateAccessTokenResponse{
		EnrollmentID: e.ID,
		CandidateID:  e.CandidateID,
		ExamID:       e.ExamID,
		EnterpriseID: e.EnterpriseID,
	}, nil
}

func (uc *sessionUseCase) StartSession(ctx context.Context, enrollmentID, enterpriseID uuid.UUID, clientIP, userAgent string, faceImage io.Reader) (*domain.ExamSession, error) {
	e, err := uc.enrollmentRepo.GetByID(ctx, enrollmentID, enterpriseID)
	if err != nil {
		return nil, err
	}

	if e.AttemptsUsed >= e.MaxAttempts {
		return nil, domain.ErrMaxAttemptsReached
	}

	if (e.TokenExpiresAt != time.Time{} && time.Now().After(e.TokenExpiresAt)) {
		return nil, domain.ErrSessionExpired
	}

	session, err := uc.sessionRepo.GetSessionByEnrollment(ctx, e.ID)
	if err != nil && err != domain.ErrSessionNotFound {
		return nil, fmt.Errorf("check active session: %w", err)
	}

	if session != nil {
		if session.Status == domain.SessionActive {
			return nil, domain.ErrSessionAlreadyActive
		}
		if session.Status == domain.SessionSubmitted {
			return nil, domain.ErrSessionAlreadySubmitted
		}

		if session.Status == domain.SessionExpired {
			return nil, domain.ErrSessionExpired
		}

		if session.Status == domain.SessionTerminated {
			// We allow, if there is attempts left. Will get back to this in future - maybe we want to show a different error if the session was terminated by system (e.g. due to cheating) vs admin action
			return nil, domain.ErrSessionTerminated
		}
	}

	// 1. Fetch Exam Metadata & validate constraints
	examCtx := logger.SetEnterpriseID(ctx, e.EnterpriseID.String())
	examMeta, err := uc.examClient.GetExamMetadata(examCtx, e.ExamID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exam metadata: %v", err)
	}

	now := time.Now().UTC()
	if examMeta.ScheduledStart == nil || now.Before(*examMeta.ScheduledStart) {
		return nil, domain.ErrExamNotScheduled
	}
	if examMeta.ScheduledEnd == nil || now.After(*examMeta.ScheduledEnd) {
		return nil, domain.ErrExamNotScheduled
	}

	// 2. Snapshot questions
	questionsMeta, err := uc.examClient.GetExamQuestions(examCtx, e.ExamID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch question snapshot: %v", err)
	}

	// 2.5 Handle Face Image Registration
	var faceURL *string
	if faceImage != nil {
		fileName := fmt.Sprintf("reg_%s_%s", e.ID.String(), uuid.New().String())
		url, err := uc.storage.Upload(ctx, fileName, faceImage)
		if err != nil {
			return nil, fmt.Errorf("failed to upload face registration image: %w", err)
		}
		faceURL = &url
	}

	// 3. Create Session and Save Question Snapshots in a transaction
	sessionID := uuid.New()
	session = &domain.ExamSession{
		ID:                sessionID,
		EnterpriseID:      e.EnterpriseID,
		ExamID:            e.ExamID,
		CandidateID:       e.CandidateID,
		EnrollmentID:      e.ID,
		Status:            domain.SessionActive,
		StartedAt:         now,
		ExpiresAt:         now.Add(time.Duration(examMeta.DurationMinutes) * time.Minute),
		ClientIP:          &clientIP,
		UserAgent:         &userAgent,
		FaceRegisteredURL: faceURL,
	}

	err = RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// 4. Mark attempt used atomically
		if err := uc.enrollmentRepo.WithTx(tx).IncrementAttempt(ctx, e.ID); err != nil {
			return fmt.Errorf("increment attempt: %w", err)
		}

		if err := uc.sessionRepo.WithTx(tx).CreateSession(ctx, session); err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		var snapshots []domain.SessionQuestion
		for _, eq := range questionsMeta {
			oIdx := 0
			if eq.OrderIndex != nil {
				oIdx = *eq.OrderIndex
			}

			// We need to snapshot the question content
			qSnapshot, err := json.Marshal(eq.Question)
			if err != nil {
				return fmt.Errorf("failed to marshal question snapshot for %s: %w", eq.QuestionID, err)
			}

			points := 0
			negativePoints := 0.0
			if eq.Question != nil {
				points = eq.Question.Points
				negativePoints = eq.Question.NegativePoints
			}

			snapshots = append(snapshots, domain.SessionQuestion{
				SessionID:        sessionID,
				QuestionID:       eq.QuestionID,
				QuestionSnapshot: qSnapshot,
				OrderIndex:       oIdx,
				Points:           points,
				NegativePoints:   negativePoints,
			})
		}

		if err := uc.sessionRepo.WithTx(tx).SaveQuestionsSnapshot(ctx, sessionID, snapshots); err != nil {
			return fmt.Errorf("save snapshots: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("StartSession transaction: %w", err)
	}

	return session, nil
}

func (uc *sessionUseCase) ResumeActiveSession(ctx context.Context, candidateID uuid.UUID) (*domain.ExamSession, error) {
	// Usually find active session by candidateID
	return nil, domain.ErrSessionNotFound
}

func (uc *sessionUseCase) GetSessionDetails(ctx context.Context, sessionID uuid.UUID, requestingUserID uuid.UUID, role string) (*domain.ExamSession, error) {
	// Add proper tenancy rules based on role
	return uc.sessionRepo.GetSessionByID(ctx, sessionID, uuid.Nil)
}

func (uc *sessionUseCase) GetSessionQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]domain.SessionQuestion, error) {
	return uc.sessionRepo.GetSessionQuestions(ctx, sessionID)
}

func (uc *sessionUseCase) SaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, questionID uuid.UUID, answerData json.RawMessage) error {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, uuid.Nil)
	if err != nil {
		return err
	}

	if session.CandidateID != candidateID {
		return domain.ErrUnauthorizedAccess
	}

	if session.Status != domain.SessionActive {
		return domain.ErrSessionNotActive
	}

	if time.Now().After(session.ExpiresAt) {
		_ = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionExpired, nil)
		return domain.ErrSessionExpired
	}

	session_question, err := uc.sessionRepo.GetSessionQuestion(ctx, sessionID, questionID)
	if err != nil {
		return err
	}



	var snapshot struct {
		Type sdomain.QuestionType `json:"type"`
	}
	if err := json.Unmarshal(session_question.QuestionSnapshot, &snapshot); err != nil {
		return fmt.Errorf("failed to decode question snapshot: %w", err)
	}

	strictUnmarshal := func(data []byte, v any) error {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		return dec.Decode(v)
	}

	switch snapshot.Type {
	case sdomain.QuestionTypeMCQ, sdomain.QuestionTypeTrueFalse:
		var a domain.MCQAnswer
		if err := strictUnmarshal(answerData, &a); err != nil {
			return domain.ErrInvalidAnswerFormat
		}
		if a.SelectedOptionIDs == nil {
			return domain.ErrInvalidAnswerFormat
		}

		// Check if the selected options are in the question snapshot
		var qSnapshot struct {
			Options []struct {
				ID string `json:"id"`
			} `json:"options"`
		}
		if err := json.Unmarshal(session_question.QuestionSnapshot, &qSnapshot); err != nil {
			return fmt.Errorf("failed to decode question snapshot for validation: %w", err)
		}
		validOptionIDs := make(map[string]bool)
		for _, opt := range qSnapshot.Options {
			validOptionIDs[opt.ID] = true
		}
		for _, selectedID := range a.SelectedOptionIDs {
			if !validOptionIDs[selectedID.String()] {
				return domain.ErrInvalidAnswerFormat
			}
		}
	case sdomain.QuestionTypeShortAnswer, sdomain.QuestionTypeEssay:
		var a domain.TextAnswer
		if err := strictUnmarshal(answerData, &a); err != nil {
			return domain.ErrInvalidAnswerFormat
		}
		if a.Text == "" {
			return domain.ErrInvalidAnswerFormat
		}
	default:
		return domain.ErrInvalidAnswerFormat
	}

	ans := &domain.SessionAnswer{
		SessionID:         sessionID,
		SessionQuestionID: session_question.ID,
		AnswerData:        answerData,
		IsFinal:           false,
	}

	return uc.sessionRepo.UpsertAnswer(ctx, ans)
}

func (uc *sessionUseCase) GetMyAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]domain.SessionAnswer, error) {
	return uc.sessionRepo.GetSessionAnswers(ctx, sessionID)
}

func (uc *sessionUseCase) SubmitExam(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, autoSubmitted bool) (*domain.ExamSubmission, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, uuid.Nil)
	if err != nil {
		return nil, err
	}

	if session.Status != domain.SessionActive {
		return nil, domain.ErrSessionNotActive
	}

	var submission *domain.ExamSubmission
	err = RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// 1. Mark term reason
		reason := "Manual Submission"
		if autoSubmitted {
			reason = "Timer Expired / Auto-submit"
		}

		if err := uc.sessionRepo.WithTx(tx).UpdateSessionStatus(ctx, sessionID, domain.SessionSubmitted, &reason); err != nil {
			return fmt.Errorf("update session status: %w", err)
		}

		// 3. Create Submission
		submission = &domain.ExamSubmission{
			SessionID:     sessionID,
			SubmittedAt:   time.Now(),
			AutoSubmitted: autoSubmitted,
		}

		if err := uc.sessionRepo.WithTx(tx).CreateSubmission(ctx, submission); err != nil {
			return fmt.Errorf("create submission: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("SubmitExam transaction: %w", err)
	}

	// Fetch candidate and exam info outside tx to avoid holding locks
	candidate, err := uc.candidateRepo.GetByID(ctx, session.CandidateID, session.EnterpriseID)
	if err != nil {
		// Log error but don't fail submission
		fmt.Printf("failed to fetch candidate for submission event: %v\n", err)
	}
	
	examCtx := logger.SetEnterpriseID(ctx, session.EnterpriseID.String())
	examMeta, err := uc.examClient.GetExamMetadata(examCtx, session.ExamID)
	if err != nil {
		fmt.Printf("failed to fetch exam metadata for submission event: %v\n", err)
	}

	// Publish event if we have candidate email
	if candidate != nil && candidate.Email != nil && examMeta != nil {
		candidateName := candidate.FirstName + " " + candidate.LastName
		event := domain.CandidateExamSubmittedEvent{
			SessionID:      sessionID,
			CandidateID:    session.CandidateID,
			ExamID:         session.ExamID,
			EnterpriseID:   session.EnterpriseID,
			CandidateName:  candidateName,
			CandidateEmail: *candidate.Email,
			ExamTitle:      examMeta.Title,
			SubmittedAt:    submission.SubmittedAt,
			AutoSubmitted:  autoSubmitted,
			Timestamp:      time.Now().UnixMilli(),
		}

		payload, err := json.Marshal(event)
		if err == nil {
			msg := messaging.Message{
				Topic: topics.CandidateExamSubmitted,
				Key:   []byte(sessionID.String()),
				Value: payload,
			}
			if err := uc.publisher.Publish(ctx, msg); err != nil {
				fmt.Printf("failed to publish submission event: %v\n", err)
			}
		} else {
			fmt.Printf("failed to marshal submission event: %v\n", err)
		}
	}

	// Publish ready for grading event asynchronously to avoid blocking
	go uc.publishReadyForGradingEvent(context.Background(), session)

	return submission, nil
}

func (uc *sessionUseCase) TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error {
	if err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionTerminated, &reason); err != nil {
		return err
	}
	
	// Fetch session to publish the event
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, enterpriseID)
	if err == nil {
		go uc.publishReadyForGradingEvent(context.Background(), session)
	}
	return nil
}

func (uc *sessionUseCase) ForceExpireSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) error {
	msg := "Admin forced expiration"
	if err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionExpired, &msg); err != nil {
		return err
	}
	return nil
}

func (uc *sessionUseCase) publishReadyForGradingEvent(ctx context.Context, session *domain.ExamSession) {
	// 1. Fetch Master Questions from Exam Service (with true evaluation criteria)
	examCtx := logger.SetEnterpriseID(ctx, session.EnterpriseID.String())
	masterQuestions, err := uc.examClient.GetExamQuestions(examCtx, session.ExamID, true)
	if err != nil {
		fmt.Printf("failed to fetch master questions for grading event: %v\n", err)
		return
	}

	masterQMap := make(map[uuid.UUID]sdomain.ExamQuestion)
	for _, q := range masterQuestions {
		masterQMap[q.QuestionID] = q
	}

	// 2. Fetch Session Questions (to link SessionQuestionID to QuestionID and get runtime points)
	sessionQuestions, err := uc.sessionRepo.GetSessionQuestions(ctx, session.ID)
	if err != nil {
		fmt.Printf("failed to fetch session questions for grading event: %v\n", err)
		return
	}

	// 3. Fetch Candidate Answers
	answers, err := uc.sessionRepo.GetSessionAnswers(ctx, session.ID)
	if err != nil {
		fmt.Printf("failed to fetch answers for grading event: %v\n", err)
		return
	}

	answerMap := make(map[uuid.UUID]domain.SessionAnswer)
	for _, a := range answers {
		answerMap[a.SessionQuestionID] = a
	}

	// 4. Construct Grading Items
	gradingItems := make([]domain.GradingItem, 0, len(sessionQuestions))
	for _, sq := range sessionQuestions {
		mq, ok := masterQMap[sq.QuestionID]
		if !ok || mq.Question == nil {
			fmt.Printf("warning: master question not found for session question %s\n", sq.ID)
			continue
		}
		item := domain.GradingItem{
			QuestionID:         sq.QuestionID,
			SessionQuestionID:  sq.ID,
			QuestionType:       string(mq.Question.Type),
			Content:            mq.Question.Content,
			Title:              mq.Question.Title,
			Topic:              mq.Question.Topic,
			MediaURL:           mq.Question.MediaURL,
			Points:             sq.Points,
			NegativePoints:     sq.NegativePoints,
			ExpectedAnswer:     mq.Question.ExpectedAnswer,
			EvaluationCriteria: mq.Question.EvaluationCriteria,
		}

		// Extract correct option IDs if applicable
		var correctOptionIDs []uuid.UUID
		for _, opt := range mq.Question.Options {
			if opt.IsCorrect {
				correctOptionIDs = append(correctOptionIDs, opt.ID)
			}
		}
		if len(correctOptionIDs) > 0 {
			item.CorrectOptionIDs = correctOptionIDs
		}

		// Attach candidate answer
		if a, hasAns := answerMap[sq.ID]; hasAns {
			item.HasAnswer = true
			var ansData domain.CandidateAnswerData
			if err := json.Unmarshal(a.AnswerData, &ansData); err == nil {
				item.CandidateAnswer = &ansData
			} else {
				fmt.Printf("warning: failed to unmarshal answer data for session question %s: %v\n", sq.ID, err)
			}
		} else {
			item.HasAnswer = false
		}

		gradingItems = append(gradingItems, item)
	}

	// Retrieve auto-submitted flag if submission exists
	autoSubmitted := false
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, session.ID, session.EnterpriseID)
	if err == nil && sub != nil {
		autoSubmitted = sub.AutoSubmitted
	}

	// 5. Construct Event
	event := domain.ExamReadyForGradingEvent{
		EventID:           uuid.New(),
		EventType:         "exam.session.ready_for_grading",
		Version:           "2.0",
		Timestamp:         time.Now(),
		EnterpriseID:      session.EnterpriseID,
		ExamID:            session.ExamID,
		SessionID:         session.ID,
		CandidateID:       session.CandidateID,
		EnrollmentID:      session.EnrollmentID,
		Status:            string(session.Status),
		StartedAt:         session.StartedAt,
		SubmittedAt:       session.SubmittedAt,
		TerminatedAt:      session.TerminatedAt,
		AutoSubmitted:     autoSubmitted,
		TerminationReason: session.TerminationReason,
		Items:             gradingItems,
	}

	payload, err := json.Marshal(event)
	if err == nil {
		msg := messaging.Message{
			Topic: topics.CandidateExamReadyForGrading,
			Key:   []byte(session.ID.String()),
			Value: payload,
		}
		if err := uc.publisher.Publish(ctx, msg); err != nil {
			fmt.Printf("failed to publish grading event: %v\n", err)
		}
	} else {
		fmt.Printf("failed to marshal grading event: %v\n", err)
	}
}
