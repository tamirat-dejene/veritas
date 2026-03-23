package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"go.uber.org/zap"
)

type sessionUseCase struct {
	pool           *pgxpool.Pool
	sessionRepo    domain.SessionRepository
	enrollmentRepo domain.EnrollmentRepository
	examClient     client.ExamServiceClient
	tokenService   domain.EnrollmentTokenService
	logger         *zap.Logger
}

func NewSessionUseCase(pool *pgxpool.Pool, sRepo domain.SessionRepository, eRepo domain.EnrollmentRepository, eClient client.ExamServiceClient, tokenService domain.EnrollmentTokenService, logger *zap.Logger) domain.SessionUseCase {
	return &sessionUseCase{
		pool:           pool,
		sessionRepo:    sRepo,
		enrollmentRepo: eRepo,
		examClient:     eClient,
		tokenService:   tokenService,
		logger:         logger,
	}
}

func (uc *sessionUseCase) ValidateAccessToken(ctx context.Context, token string) (*domain.ValidateAccessTokenResponse, error) {
	claims, err := uc.tokenService.ParseToken(ctx, token)
	if err != nil {
		return nil, domain.ErrInvalidAccessToken
	}

	return &domain.ValidateAccessTokenResponse{
		EnrollmentID: claims.EnrollmentID,
		CandidateID:  claims.CandidateID,
		ExamID:       claims.ExamID,
		EnterpriseID: claims.EnterpriseID,
	}, nil
}

func (uc *sessionUseCase) StartSession(ctx context.Context, token string, clientIP, userAgent string) (*domain.ExamSession, error) {
	claims, err := uc.tokenService.ParseToken(ctx, token)
	if err != nil {
		return nil, domain.ErrInvalidAccessToken
	}

	e, err := uc.enrollmentRepo.GetByID(ctx, claims.EnrollmentID, claims.EnterpriseID)
	if err != nil {
		return nil, err
	}

	// Verify that the token presented matches the one current in DB (allows revocation/rotation)
	if e.AccessTokenHash != HashToken(token) {
		return nil, domain.ErrInvalidAccessToken
	}

	if e.AttemptsUsed >= e.MaxAttempts {
		return nil, domain.ErrMaxAttemptsReached
	}

	// 2. Fetch Exam Metadata & validate constraints
	examMeta, err := uc.examClient.GetExamMetadata(ctx, e.ExamID, e.EnterpriseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exam metadata: %v", err)
	}

	now := time.Now()
	if examMeta.ScheduledStart != nil && now.Before(*examMeta.ScheduledStart) {
		return nil, domain.ErrExamNotScheduled
	}
	if examMeta.ScheduledEnd != nil && now.After(*examMeta.ScheduledEnd) {
		return nil, domain.ErrExamNotScheduled
	}

	// 3. Mark attempt used atomically (Mocked conceptually via Update)
	if err := uc.enrollmentRepo.IncrementAttempt(ctx, e.ID); err != nil {
		return nil, err
	}

	// 4. Snapshot questions
	questionsMeta, err := uc.examClient.GetExamQuestions(ctx, e.ExamID, e.EnterpriseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch question snapshot: %v", err)
	}

	// 5. Create Session and Save Question Snapshots in a transaction
	sessionID := uuid.New()
	session := &domain.ExamSession{
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
		// 3. Mark attempt used atomically
		if err := uc.enrollmentRepo.WithTx(tx).IncrementAttempt(ctx, e.ID); err != nil {
			return fmt.Errorf("increment attempt: %w", err)
		}

		if err := uc.sessionRepo.WithTx(tx).CreateSession(ctx, session); err != nil {
			return fmt.Errorf("create session: %w", err)
		}

		var snapshots []domain.SessionQuestion
		for _, qm := range questionsMeta {
			oReq := 0
			if qm.OrderIndex != nil {
				oReq = *qm.OrderIndex
			}

			snapshots = append(snapshots, domain.SessionQuestion{
				SessionID:        sessionID,
				QuestionID:       qm.ID,
				QuestionSnapshot: qm.Content,
				OrderIndex:       oReq,
				Points:           qm.Points,
				NegativePoints:   qm.NegativePoints,
			})
		}

		if err := uc.sessionRepo.WithTx(tx).SaveQuestionsSnapshot(ctx, sessionID, snapshots); err != nil {
			return fmt.Errorf("save snapshots: %w", err)
		}
		return nil
	})

	if err != nil {
		uc.logger.Error("session start failed", zap.Error(err), zap.String("enrollmentID", e.ID.String()))
		return nil, fmt.Errorf("StartSession transaction: %w", err)
	}

	uc.logger.Info("session started", zap.String("sessionID", session.ID.String()), zap.String("candidateID", session.CandidateID.String()), zap.String("examID", session.ExamID.String()))
	return session, nil
}

func (uc *sessionUseCase) ResumeActiveSession(ctx context.Context, candidateID uuid.UUID) (*domain.ExamSession, error) {
	// Usually find active session by candidateID
	return nil, domain.ErrSessionNotFound
}

func (uc *sessionUseCase) GetSessionDetails(ctx context.Context, sessionID uuid.UUID, requestingUserID uuid.UUID, role string) (*domain.ExamSession, error) {
	// Add proper tenancy rules based on role
	return uc.sessionRepo.GetSessionByID(ctx, sessionID)
}

func (uc *sessionUseCase) GetSessionQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]domain.SessionQuestion, error) {
	return uc.sessionRepo.GetSessionQuestions(ctx, sessionID)
}

func (uc *sessionUseCase) SaveAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, questionID uuid.UUID, answerData json.RawMessage) error {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != domain.SessionActive {
		return domain.ErrSessionNotActive
	}
	if time.Now().After(session.ExpiresAt) {
		_ = uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionExpired, nil)
		uc.logger.Info("session expired", zap.String("sessionID", sessionID.String()))
		return domain.ErrSessionExpired
	}

	ans := &domain.SessionAnswer{
		SessionID:         sessionID,
		SessionQuestionID: questionID,
		AnswerData:        answerData,
		IsFinal:           false,
	}

	return uc.sessionRepo.UpsertAnswer(ctx, ans)
}

func (uc *sessionUseCase) GetMyAnswers(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) ([]domain.SessionAnswer, error) {
	return uc.sessionRepo.GetSessionAnswers(ctx, sessionID)
}

func (uc *sessionUseCase) SubmitExam(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID, autoSubmitted bool) (*domain.ExamSubmission, error) {
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID)
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
		uc.logger.Error("submission failed", zap.Error(err), zap.String("sessionID", sessionID.String()))
		return nil, fmt.Errorf("SubmitExam transaction: %w", err)
	}

	uc.logger.Info("exam submitted", zap.String("sessionID", sessionID.String()), zap.Bool("autoSubmitted", autoSubmitted))

	// Fetch the created submission to return it (or we could just use the one we built if we assigned an ID)
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetch created submission: %w", err)
	}

	return sub, nil
}

func (uc *sessionUseCase) TerminateSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID, reason string) error {
	if err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionTerminated, &reason); err != nil {
		return err
	}
	uc.logger.Warn("session terminated", zap.String("sessionID", sessionID.String()), zap.String("reason", reason))
	return nil
}

func (uc *sessionUseCase) ForceExpireSession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) error {
	msg := "Admin forced expiration"
	if err := uc.sessionRepo.UpdateSessionStatus(ctx, sessionID, domain.SessionExpired, &msg); err != nil {
		return err
	}
	uc.logger.Warn("session force-expired", zap.String("sessionID", sessionID.String()))
	return nil
}
