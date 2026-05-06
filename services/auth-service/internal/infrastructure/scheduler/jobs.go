package scheduler

import (
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/cronjob"
)

func RegisterAuthJobs(s *cronjob.Scheduler, refreshUC *usecase.RefreshUseCase) {
	s.Register(
		cronjob.Job{
			Name:     "purge-expired-tokens",
			Schedule: "@daily",
			Fn:       refreshUC.PurgeExpiredTokens,
		},
		cronjob.Job{
			Name:     "audit-session-integrity",
			Schedule: "@daily",
			Fn:       refreshUC.AuditSessionIntegrity,
		},
	)
}
