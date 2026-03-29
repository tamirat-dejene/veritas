package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
)

type monitoringUseCase struct {
	sessionRepo domain.SessionRepository
	logger      *zap.Logger
}

func NewMonitoringUseCase(sRepo domain.SessionRepository, logger *zap.Logger) domain.MonitoringUseCase {
	return &monitoringUseCase{
		sessionRepo: sRepo,
		logger:      logger,
	}
}

func (uc *monitoringUseCase) ListSessionsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, status *domain.SessionStatus, candidateID *uuid.UUID, params pagination.Params) ([]*domain.ExamSession, int64, error) {
	return uc.sessionRepo.ListSessionsByExam(ctx, examID, enterpriseID, status, params)
}

func (uc *monitoringUseCase) GetSessionSummary(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamSession, error) {
	return uc.sessionRepo.GetSessionByID(ctx, sessionID, enterpriseID)
}

func (uc *monitoringUseCase) GetSubmissions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamSubmission, int64, error) {
	return uc.sessionRepo.GetSubmissionsByExam(ctx, examID, enterpriseID, params)
}

func (uc *monitoringUseCase) GetSubmissionDetail(ctx context.Context, submissionID uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamSubmission, error) {
	return uc.sessionRepo.GetSubmissionByID(ctx, submissionID, enterpriseID)
}

func (uc *monitoringUseCase) CandidateGetResult(ctx context.Context, sessionID uuid.UUID, candidateID uuid.UUID) (*domain.ExamSubmission, error) {
	// 1. Fetch session to verify ownership and get enterpriseID
	session, err := uc.sessionRepo.GetSessionByID(ctx, sessionID, uuid.Nil)
	if err != nil {
		return nil, err
	}

	if session.CandidateID != candidateID {
		return nil, domain.ErrUnauthorizedAccess
	}

	// 2. Fetch submission
	sub, err := uc.sessionRepo.GetSubmissionBySession(ctx, sessionID, session.EnterpriseID)
	if err != nil {
		return nil, err
	}

	// 3. Check grading status release policy
	if sub.GradingStatus != "Released" {
		return nil, domain.ErrUnauthorizedAccess
	}

	return sub, nil
}
