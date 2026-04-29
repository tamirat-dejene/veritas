package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
)

type sessionUseCase struct {
	pool           *pgxpool.Pool
	sessionRepo    domain.SessionRepository
	enrollmentRepo domain.EnrollmentRepository
	examClient     client.ExamServiceClient
	tokenService   domain.EnrollmentTokenService
}

func NewSessionUseCase(pool *pgxpool.Pool, sRepo domain.SessionRepository, eRepo domain.EnrollmentRepository, eClient client.ExamServiceClient, tokenService domain.EnrollmentTokenService) domain.SessionUseCase {
	return &sessionUseCase{
		pool:           pool,
		sessionRepo:    sRepo,
		enrollmentRepo: eRepo,
		examClient:     eClient,
		tokenService:   tokenService,
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

func (uc *sessionUseCase) StartSession(ctx context.Context, enrollmentID, enterpriseID uuid.UUID, clientIP, userAgent string) (*domain.ExamSession, error) {
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
	questionsMeta, err := uc.examClient.GetExamQuestions(examCtx, e.ExamID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch question snapshot: %v", err)
	}

	// 3. Create Session and Save Question Snapshots in a transaction
	sessionID := uuid.New()
	session = &domain.ExamSession{
		ID:           sessionID,
		EnterpriseID: e.EnterpriseID,
		ExamID:       e.ExamID,
		CandidateID:  e.CandidateID,
		EnrollmentID: e.ID,
		Status:       domain.SessionActive,
		StartedAt:    now,
		ExpiresAt:    now.Add(time.Duration(examMeta.DurationMinutes) * time.Minute),
		ClientIP:     &clientIP,
		UserAgent:    &userAgent,
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
			if eq.PointsOverride != nil {
				points = *eq.PointsOverride
			} else if eq.Question != nil {
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
		submission := &domain.ExamSubmission{
			SessionID:     sessionID,
			SubmittedAt:   time.Now(),
			AutoSubmitted: autoSubmitted,
			GradingStatus: "Pending", // Or 'ReadyForGrading'
		}

		if err := uc.sessionRepo.WithTx(tx).CreateSubmission(ctx, submission); err != nil {
			return fmt.Errorf("create submission: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("SubmitExam transaction: %w", err)
	}

	// Fetch the created submission to return it (or we could just use the one we built if we assigned an ID)
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, sessionID, session.EnterpriseID)
	if err != nil {
		return nil, fmt.Errorf("fetch created submission: %w", err)
	}

	return sub, nil
}

func (uc *sessionUseCase) TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error {
	if err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionTerminated, &reason); err != nil {
		return err
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
