package scheduler

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
)

// RegisterExamJobs registers background maintenance tasks for the exam service.
func RegisterExamJobs(scheduler *cronjob.Scheduler, maintenanceUC domain.MaintenanceUseCase) {
	scheduler.Register(
		cronjob.Job{
			Name:     "publish_scheduled_exams",
			Schedule: "* * * * *", // Every minute
			Fn: func(ctx context.Context) error {
				return maintenanceUC.PublishScheduledExams(ctx)
			},
		},
		cronjob.Job{
			Name:     "close_expired_exams",
			Schedule: "* * * * *", // Every minute
			Fn: func(ctx context.Context) error {
				return maintenanceUC.CloseExpiredExams(ctx)
			},
		},
		cronjob.Job{
			Name:     "archive_stale_exams",
			Schedule: "0 2 * * *", // Daily at 2 AM
			Fn: func(ctx context.Context) error {
				return maintenanceUC.ArchiveStaleExams(ctx)
			},
		},
	)
}
