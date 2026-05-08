package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

type consistencyUseCase struct {
	sessionRepo    domain.SessionRepository
	enrollmentRepo domain.EnrollmentRepository
	candidateRepo  domain.CandidateRepository
	log            *zap.Logger
}

// NewConsistencyUseCase creates a new instance of ConsistencyUseCase.
func NewConsistencyUseCase(
	sessionRepo domain.SessionRepository,
	enrollmentRepo domain.EnrollmentRepository,
	candidateRepo domain.CandidateRepository,
	log *zap.Logger,
) domain.ConsistencyUseCase {
	return &consistencyUseCase{
		sessionRepo:    sessionRepo,
		enrollmentRepo: enrollmentRepo,
		candidateRepo:  candidateRepo,
		log:            log,
	}
}

// HandleEnterpriseDeactivated responds to an enterprise being suspended or deleted.
func (uc *consistencyUseCase) HandleEnterpriseDeactivated(ctx context.Context, enterpriseID uuid.UUID) error {
	l := logger.WithContext(ctx, uc.log).With(zap.String("enterprise_id", enterpriseID.String()))

	// 1. Terminate all active sessions
	if err := uc.sessionRepo.TerminateActiveSessionsByEnterprise(ctx, enterpriseID, "Enterprise Suspended/Deleted"); err != nil {
		l.Error("failed to terminate active sessions for enterprise", zap.Error(err))
	}

	// 2. Revoke all enrollments
	if err := uc.enrollmentRepo.RevokeByEnterprise(ctx, enterpriseID); err != nil {
		l.Error("failed to revoke enrollments for enterprise", zap.Error(err))
	}

	// 3. Deactivate all candidate profiles
	if err := uc.candidateRepo.DeactivateByEnterprise(ctx, enterpriseID); err != nil {
		l.Error("failed to deactivate candidates for enterprise", zap.Error(err))
	}

	return nil
}

// HandleExamClosed responds to an exam being closed or deleted.
func (uc *consistencyUseCase) HandleExamClosed(ctx context.Context, examID uuid.UUID) error {
	l := logger.WithContext(ctx, uc.log).With(zap.String("exam_id", examID.String()))

	// 1. Terminate all active sessions for the exam
	if err := uc.sessionRepo.TerminateActiveSessionsByExam(ctx, examID, "Exam Closed/Deleted"); err != nil {
		l.Error("failed to terminate active sessions for exam", zap.Error(err))
	}

	// 2. Revoke all enrollments for the exam
	if err := uc.enrollmentRepo.RevokeByExam(ctx, examID); err != nil {
		l.Error("failed to revoke enrollments for exam", zap.Error(err))
	}

	return nil
}
