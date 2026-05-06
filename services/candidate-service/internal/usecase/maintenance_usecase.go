package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

type maintenanceUseCase struct {
	sessionRepo    domain.SessionRepository
	sessionUC      domain.SessionUseCase
	enrollmentRepo domain.EnrollmentRepository
	log            *zap.Logger
}

func NewMaintenanceUseCase(
	sessionRepo domain.SessionRepository,
	sessionUC domain.SessionUseCase,
	enrollmentRepo domain.EnrollmentRepository,
	log *zap.Logger,
) domain.MaintenanceUseCase {
	return &maintenanceUseCase{
		sessionRepo:    sessionRepo,
		sessionUC:      sessionUC,
		enrollmentRepo: enrollmentRepo,
		log:            log.Named("maintenance_usecase"),
	}
}

func (uc *maintenanceUseCase) ProcessExpiredSessions(ctx context.Context) error {
	log := logger.WithContext(ctx, uc.log).With(zap.String("task", "ProcessExpiredSessions"))

	// Process in batches
	batchSize := 100
	sessions, err := uc.sessionRepo.GetExpiredActiveSessions(ctx, batchSize)
	if err != nil {
		log.Error("failed to fetch expired active sessions", zap.Error(err))
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	log.Info("processing expired active sessions", zap.Int("count", len(sessions)))

	successCount := 0
	for _, session := range sessions {
		// Use uuid.Nil for candidateID as it's a system action
		_, err := uc.sessionUC.SubmitExam(ctx, session.ID, uuid.Nil, true)
		if err != nil {
			log.Error("failed to auto-submit expired session",
				zap.String("session_id", session.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	log.Info("completed processing expired sessions",
		zap.Int("total", len(sessions)),
		zap.Int("success", successCount),
		zap.Int("failed", len(sessions)-successCount))

	return nil
}

func (uc *maintenanceUseCase) ProcessExpiredEnrollments(ctx context.Context) error {
	log := logger.WithContext(ctx, uc.log).With(zap.String("task", "ProcessExpiredEnrollments"))

	batchSize := 100
	enrollments, err := uc.enrollmentRepo.GetExpiredPendingEnrollments(ctx, batchSize)
	if err != nil {
		log.Error("failed to fetch expired pending enrollments", zap.Error(err))
		return err
	}

	if len(enrollments) == 0 {
		return nil
	}

	log.Info("processing expired pending enrollments", zap.Int("count", len(enrollments)))

	successCount := 0
	for _, enrollment := range enrollments {
		if err := uc.enrollmentRepo.UpdateStatus(ctx, enrollment.ID, domain.StatusRevoked); err != nil {
			log.Error("failed to revoke expired enrollment",
				zap.String("enrollment_id", enrollment.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	log.Info("completed processing expired enrollments",
		zap.Int("total", len(enrollments)),
		zap.Int("success", successCount),
		zap.Int("failed", len(enrollments)-successCount))

	return nil
}
