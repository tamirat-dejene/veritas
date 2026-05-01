package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type monitoringUseCase struct {
	sessionRepo domain.SessionRepository
}

func NewMonitoringUseCase(sRepo domain.SessionRepository) domain.MonitoringUseCase {
	return &monitoringUseCase{
		sessionRepo: sRepo,
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
