package scheduler

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
)

// RegisterCandidateJobs registers background maintenance tasks for the candidate service.
func RegisterCandidateJobs(scheduler *cronjob.Scheduler, maintenanceUC domain.MaintenanceUseCase) {
	scheduler.Register(
		cronjob.Job{
			Name:     "auto_submit_sessions",
			Schedule: "* * * * *", // Every minute
			Fn: func(ctx context.Context) error {
				return maintenanceUC.ProcessExpiredSessions(ctx)
			},
		},
		cronjob.Job{
			Name:     "revoke_expired_enrollments",
			Schedule: "0 * * * *", // Hourly
			Fn: func(ctx context.Context) error {
				return maintenanceUC.ProcessExpiredEnrollments(ctx)
			},
		},
	)
}
