package scheduler

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
)

// RegisterPaymentJobs registers background maintenance tasks for the payment service.
func RegisterPaymentJobs(scheduler *cronjob.Scheduler, maintenanceUC domain.MaintenanceUseCase) {
	scheduler.Register(
		cronjob.Job{
			Name:     "expire_lapsed_subscriptions",
			Schedule: "*/30 * * * *", // Every 30 minutes
			Fn: func(ctx context.Context) error {
				return maintenanceUC.ExpireLapsedSubscriptions(ctx)
			},
		},
		cronjob.Job{
			Name:     "mark_overdue_invoices",
			Schedule: "0 * * * *", // Hourly
			Fn: func(ctx context.Context) error {
				return maintenanceUC.MarkOverdueInvoicesUncollectible(ctx)
			},
		},
		cronjob.Job{
			Name:     "purge_webhook_events",
			Schedule: "0 3 * * *", // Daily at 3 AM
			Fn: func(ctx context.Context) error {
				return maintenanceUC.PurgeOldWebhookEvents(ctx)
			},
		},
	)
}
