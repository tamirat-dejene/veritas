package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"go.uber.org/zap"
)

// enterpriseLifecycleEvent is the JSON payload published by enterprise-service
// for enterprise-level lifecycle events.
type enterpriseLifecycleEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

// NewEventConsumer creates a messaging.Router configured to handle incoming
// enterprise lifecycle events and drive consistency within the exam-service.
//
// Subscribed topics:
//   - enterprise.enterprise.suspended  → force-close active exams, cancel scheduled
//   - enterprise.enterprise.deleted    → same as suspended (soft-delete)
//   - enterprise.enterprise.hard_deleted → above + archive closed exams
func NewEventConsumer(usecase domain.ConsistencyUseCase, logger *zap.Logger) *messaging.Router {
	router := messaging.NewRouter(loggingMiddleware(logger))

	// Both suspended and soft-deleted enterprises trigger the same reaction.
	suspendedHandler := func(ctx context.Context, evt enterpriseLifecycleEvent) error {
		return usecase.HandleEnterpriseSuspended(ctx, evt.EnterpriseID)
	}
	messaging.RegisterJSONHandler(router, topics.EnterpriseSuspended, suspendedHandler)
	messaging.RegisterJSONHandler(router, topics.EnterpriseDeleted, suspendedHandler)

	// Hard-delete additionally archives any remaining closed exams.
	messaging.RegisterJSONHandler(router, topics.EnterpriseHardDeleted,
		func(ctx context.Context, evt enterpriseLifecycleEvent) error {
			return usecase.HandleEnterpriseHardDeleted(ctx, evt.EnterpriseID)
		},
	)

	return router
}

// loggingMiddleware is a router-level middleware that logs every dispatched
// message and any processing errors. Mirrors the pattern used in auth-service
// and enterprise-service consumers.
func loggingMiddleware(logger *zap.Logger) messaging.Middleware {
	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg messaging.Message) error {
			logger.Debug("processing kafka message for consistency",
				zap.String("topic", msg.Topic),
				zap.Int("payload_size", len(msg.Value)),
			)
			err := next(ctx, msg)
			if err != nil {
				logger.Error("failed to process kafka message for consistency",
					zap.String("topic", msg.Topic),
					zap.Error(err),
				)
			}
			return err
		}
	}
}
