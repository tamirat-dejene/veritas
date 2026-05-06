package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"go.uber.org/zap"
)

type maintenanceUseCase struct {
	userRepo          domain.UserRepository
	passwordResetRepo domain.PasswordResetRepository
	enterpriseRepo    domain.EnterpriseRepository
	enterpriseUC      domain.EnterpriseUsecase
	auditRepo         domain.AuditRepository
	log               *zap.Logger
}

func NewMaintenanceUseCase(
	userRepo domain.UserRepository,
	passwordResetRepo domain.PasswordResetRepository,
	enterpriseRepo domain.EnterpriseRepository,
	enterpriseUC domain.EnterpriseUsecase,
	auditRepo domain.AuditRepository,
	log *zap.Logger,
) domain.MaintenanceUseCase {
	return &maintenanceUseCase{
		userRepo:          userRepo,
		passwordResetRepo: passwordResetRepo,
		enterpriseRepo:    enterpriseRepo,
		enterpriseUC:      enterpriseUC,
		auditRepo:         auditRepo,
		log:               log.Named("maintenance_usecase"),
	}
}

func (uc *maintenanceUseCase) PurgeExpiredPasswordResetTokens(ctx context.Context) error {
	count, err := uc.passwordResetRepo.DeleteExpiredTokens(ctx)
	if err != nil {
		uc.log.Error("failed to purge expired password reset tokens", zap.Error(err))
		return err
	}
	if count > 0 {
		uc.log.Info("purged expired password reset tokens", zap.Int64("count", count))
	}
	return nil
}

func (uc *maintenanceUseCase) HardDeleteExpiredEnterprises(ctx context.Context) error {
	batchSize := 50
	enterprises, err := uc.enterpriseRepo.GetExpiredDeletedEnterprises(ctx, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch expired deleted enterprises", zap.Error(err))
		return err
	}

	if len(enterprises) == 0 {
		return nil
	}

	uc.log.Info("processing expired deleted enterprises for hard deletion", zap.Int("count", len(enterprises)))

	successCount := 0
	for _, e := range enterprises {
		// Use uuid.Nil for adminID since this is an automated system task
		if err := uc.enterpriseUC.HardDeleteEnterprise(ctx, e.ID, uuid.Nil); err != nil {
			uc.log.Error("failed to hard delete enterprise",
				zap.String("enterprise_id", e.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	uc.log.Info("completed hard delete of expired enterprises",
		zap.Int("total", len(enterprises)),
		zap.Int("success", successCount))

	return nil
}

func (uc *maintenanceUseCase) ResetExpiredAccountLocks(ctx context.Context) error {
	count, err := uc.userRepo.ResetExpiredLocks(ctx)
	if err != nil {
		uc.log.Error("failed to reset expired account locks", zap.Error(err))
		return err
	}
	if count > 0 {
		uc.log.Info("reset expired account locks", zap.Int64("count", count))
	}
	return nil
}

func (uc *maintenanceUseCase) PurgeOldAuditLogs(ctx context.Context) error {
	// Retention period: 90 days as approved by the user
	retentionDays := 90
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	count, err := uc.auditRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		uc.log.Error("failed to purge old audit logs", zap.Error(err))
		return err
	}
	if count > 0 {
		uc.log.Info("purged old audit logs", zap.Int64("count", count), zap.Time("cutoff", cutoff))
	}
	return nil
}
