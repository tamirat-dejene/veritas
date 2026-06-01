package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"encoding/base64"
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
	faceAPIURL     string
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
	faceAPIURL string,
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
		faceAPIURL:     faceAPIURL,
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
	examMeta, err := uc.examClient.GetExamMetadata(ctx, e.EnterpriseID, e.ExamID)
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
	questionsMeta, err := uc.examClient.GetExamQuestions(ctx, e.EnterpriseID, e.ExamID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch question snapshot: %v", err)
	}

	// 2.5 Handle Face Image Registration
	var faceURL *string
	var faceEmbedding []float64
	if faceImage != nil {
		imgData, err := io.ReadAll(faceImage)
		if err != nil {
			return nil, fmt.Errorf("failed to read face registration image: %w", err)
		}

		fileName := fmt.Sprintf("reg_%s_%s", e.ID.String(), uuid.New().String())
		url, err := uc.storage.Upload(ctx, fileName, bytes.NewReader(imgData))
		if err != nil {
			return nil, fmt.Errorf("failed to upload face registration image: %w", err)
		}
		faceURL = &url

		if uc.faceAPIURL != "" {
			b64Str := base64.StdEncoding.EncodeToString(imgData)
			payloadBytes, _ := json.Marshal(map[string]interface{}{
				"img": b64Str,
				"model_name": "Facenet512",
				"detector_backend": "retinaface",
			})
			req, _ := http.NewRequestWithContext(ctx, "POST", uc.faceAPIURL+"/embed", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			
			httpClient := &http.Client{Timeout: 15 * time.Second}
			resp, err := httpClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var embedResp struct {
						Embedding []float64 `json:"embedding"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&embedResp); err == nil {
						faceEmbedding = embedResp.Embedding
					}
				} else {
					fmt.Printf("warning: face embedding failed with status %d\n", resp.StatusCode)
				}
			} else {
				fmt.Printf("warning: face embedding request failed: %v\n", err)
			}
		}
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
		FaceRegisteredEmbedding: faceEmbedding,
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

func (uc *sessionUseCase) BulkSaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, items []domain.BulkAnswerItem) ([]domain.BulkAnswerResult, error) {
	// 1. Validate session once (ownership, status, expiry)
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if session.CandidateID != candidateID {
		return nil, domain.ErrUnauthorizedAccess
	}
	if session.Status != domain.SessionActive {
		return nil, domain.ErrSessionNotActive
	}
	if time.Now().After(session.ExpiresAt) {
		_ = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionExpired, nil)
		return nil, domain.ErrSessionExpired
	}

	// 2. Load all session questions once into a map for O(1) look-up
	sessionQuestions, err := uc.sessionRepo.GetSessionQuestions(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session questions: %w", err)
	}
	questionMap := make(map[uuid.UUID]domain.SessionQuestion, len(sessionQuestions))
	for _, sq := range sessionQuestions {
		questionMap[sq.ID] = sq
	}

	strictUnmarshal := func(data []byte, v any) error {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		return dec.Decode(v)
	}

	// 3. Validate each item and accumulate results
	results := make([]domain.BulkAnswerResult, 0, len(items))
	validAnswers := make([]*domain.SessionAnswer, 0, len(items))

	for _, item := range items {
		result := domain.BulkAnswerResult{
			SessionQuestionID: item.SessionQuestionID,
			Status:            "saved",
		}

		sq, ok := questionMap[item.SessionQuestionID]
		if !ok {
			msg := domain.ErrQuestionNotFound.Error()
			result.Status = "failed"
			result.Error = &msg
			results = append(results, result)
			continue
		}

		var snapshot struct {
			Type sdomain.QuestionType `json:"type"`
		}
		if err := json.Unmarshal(sq.QuestionSnapshot, &snapshot); err != nil {
			msg := "failed to decode question snapshot"
			result.Status = "failed"
			result.Error = &msg
			results = append(results, result)
			continue
		}

		var validationErr error
		switch snapshot.Type {
		case sdomain.QuestionTypeMCQ, sdomain.QuestionTypeTrueFalse:
			var a domain.MCQAnswer
			if err := strictUnmarshal(item.AnswerData, &a); err != nil || a.SelectedOptionIDs == nil {
				validationErr = domain.ErrInvalidAnswerFormat
				break
			}
			var qSnapshot struct {
				Options []struct {
					ID string `json:"id"`
				} `json:"options"`
			}
			if err := json.Unmarshal(sq.QuestionSnapshot, &qSnapshot); err != nil {
				validationErr = domain.ErrInvalidAnswerFormat
				break
			}
			validOptionIDs := make(map[string]bool, len(qSnapshot.Options))
			for _, opt := range qSnapshot.Options {
				validOptionIDs[opt.ID] = true
			}
			for _, selectedID := range a.SelectedOptionIDs {
				if !validOptionIDs[selectedID.String()] {
					validationErr = domain.ErrInvalidAnswerFormat
					break
				}
			}
		case sdomain.QuestionTypeShortAnswer, sdomain.QuestionTypeEssay:
			var a domain.TextAnswer
			if err := strictUnmarshal(item.AnswerData, &a); err != nil || a.Text == "" {
				validationErr = domain.ErrInvalidAnswerFormat
			}
		default:
			validationErr = domain.ErrInvalidAnswerFormat
		}

		if validationErr != nil {
			msg := validationErr.Error()
			result.Status = "failed"
			result.Error = &msg
			results = append(results, result)
			continue
		}

		validAnswers = append(validAnswers, &domain.SessionAnswer{
			SessionID:         sessionID,
			SessionQuestionID: sq.ID,
			AnswerData:        item.AnswerData,
			IsFinal:           false,
		})
		results = append(results, result) // optimistic "saved"; may be overwritten below
	}

	// 4. Persist valid answers; mark DB failures in results
	if len(validAnswers) > 0 {
		failedIDs, _ := uc.sessionRepo.BulkUpsertAnswer(ctx, validAnswers)
		if len(failedIDs) > 0 {
			failedSet := make(map[uuid.UUID]bool, len(failedIDs))
			for _, id := range failedIDs {
				failedSet[id] = true
			}
			dbErr := "failed to persist answer"
			for i, r := range results {
				if r.Status == "saved" && failedSet[r.SessionQuestionID] {
					results[i].Status = "failed"
					results[i].Error = &dbErr
				}
			}
		}
	}

	return results, nil
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
	
	examMeta, err := uc.examClient.GetExamMetadata(ctx, session.EnterpriseID, session.ExamID)
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
	// Retrieve auto-submitted flag if submission exists
	autoSubmitted := false
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, session.ID, session.EnterpriseID)
	if err == nil && sub != nil {
		autoSubmitted = sub.AutoSubmitted
	}

	// Publish the slim trigger event — grading-service will pull the full payload via HTTP.
	event := domain.ExamReadyForGradingEvent{
		EventID:           uuid.New(),
		EventType:         topics.ExamSessionReadyForGrading,
		Version:           "3.0",
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
	}

	payload, err := json.Marshal(event)
	if err == nil {
		msg := messaging.Message{
			Topic: topics.ExamSessionReadyForGrading,
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

// BuildGradingPayload assembles the full grading data for a session: it fetches
// master questions (with evaluation criteria) from the exam-service, loads the
// candidate's session question snapshots and answers from the local repository,
// and zips them into a GradingPayload ready for the grading-service to consume.
func (uc *sessionUseCase) BuildGradingPayload(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*domain.GradingPayload, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// 1. Fetch master questions with true evaluation criteria from exam-service
	masterQuestions, err := uc.examClient.GetExamQuestions(ctx, session.EnterpriseID, session.ExamID, true)
	if err != nil {
		return nil, fmt.Errorf("fetch master questions: %w", err)
	}

	masterQMap := make(map[uuid.UUID]sdomain.ExamQuestion, len(masterQuestions))
	for _, q := range masterQuestions {
		masterQMap[q.QuestionID] = q
	}

	// 2. Fetch session question snapshots
	sessionQuestions, err := uc.sessionRepo.GetSessionQuestions(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch session questions: %w", err)
	}

	// 3. Fetch candidate answers
	answers, err := uc.sessionRepo.GetSessionAnswers(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch session answers: %w", err)
	}

	answerMap := make(map[uuid.UUID]domain.SessionAnswer, len(answers))
	for _, a := range answers {
		answerMap[a.SessionQuestionID] = a
	}

	// 4. Construct grading items
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

		var correctOptionIDs []uuid.UUID
		for _, opt := range mq.Question.Options {
			if opt.IsCorrect {
				correctOptionIDs = append(correctOptionIDs, opt.ID)
			}
		}
		if len(correctOptionIDs) > 0 {
			item.CorrectOptionIDs = correctOptionIDs
		}

		if len(mq.Question.Options) > 0 {
			opts := make([]domain.GradingOption, 0, len(mq.Question.Options))
			for _, opt := range mq.Question.Options {
				opts = append(opts, domain.GradingOption{
					ID:      opt.ID,
					Content: opt.Content,
				})
			}
			item.Options = opts
		}

		if a, hasAns := answerMap[sq.ID]; hasAns {
			item.HasAnswer = true
			var ansData domain.CandidateAnswerData
			if err := json.Unmarshal(a.AnswerData, &ansData); err == nil {
				item.CandidateAnswer = &ansData
			} else {
				fmt.Printf("warning: failed to unmarshal answer for session question %s: %v\n", sq.ID, err)
			}
		}

		gradingItems = append(gradingItems, item)
	}

	// 5. Resolve auto-submitted flag
	autoSubmitted := false
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, session.ID, session.EnterpriseID)
	if err == nil && sub != nil {
		autoSubmitted = sub.AutoSubmitted
	}

	return &domain.GradingPayload{
		SessionID:         session.ID,
		EnterpriseID:      session.EnterpriseID,
		ExamID:            session.ExamID,
		CandidateID:       session.CandidateID,
		EnrollmentID:      session.EnrollmentID,
		Status:            string(session.Status),
		StartedAt:         session.StartedAt,
		SubmittedAt:       session.SubmittedAt,
		TerminatedAt:      session.TerminatedAt,
		AutoSubmitted:     autoSubmitted,
		TerminationReason: session.TerminationReason,
		Items:             gradingItems,
	}, nil
}
