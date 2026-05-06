package scheduler

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
)

// RegisterEnterpriseJobs registers background maintenance tasks for the enterprise service.
func RegisterEnterpriseJobs(scheduler *cronjob.Scheduler, maintenanceUC domain.MaintenanceUseCase) {
	scheduler.Register(
		cronjob.Job{
			Name:     "purge_password_tokens",
			Schedule: "0 * * * *", // Hourly
			Fn: func(ctx context.Context) error {
				return maintenanceUC.PurgeExpiredPasswordResetTokens(ctx)
			},
		},
		cronjob.Job{
			Name:     "hard_delete_enterprises",
			Schedule: "0 0 * * *", // Daily at midnight
			Fn: func(ctx context.Context) error {
				return maintenanceUC.HardDeleteExpiredEnterprises(ctx)
			},
		},
		cronjob.Job{
			Name:     "reset_account_locks",
			Schedule: "*/5 * * * *", // Every 5 minutes
			Fn: func(ctx context.Context) error {
				return maintenanceUC.ResetExpiredAccountLocks(ctx)
			},
		},
		cronjob.Job{
			Name:     "purge_audit_logs",
			Schedule: "0 1 * * *", // Daily at 1 AM
			Fn: func(ctx context.Context) error {
				return maintenanceUC.PurgeOldAuditLogs(ctx)
			},
		},
	)
}
