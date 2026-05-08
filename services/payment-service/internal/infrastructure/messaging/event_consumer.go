package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
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
// enterprise hard-deletion events and clean up billing data in the payment-service.
//
// Subscribed topics:
//   - enterprise.enterprise.hard_deleted → cancel subscription + void open invoices
func NewEventConsumer(usecase domain.ConsistencyUseCase, logger *zap.Logger) *messaging.Router {
	router := messaging.NewRouter(loggingMiddleware(logger))

	messaging.RegisterJSONHandler(router, topics.EnterpriseHardDeleted,
		func(ctx context.Context, evt enterpriseLifecycleEvent) error {
			return usecase.HandleEnterpriseHardDeleted(ctx, evt.EnterpriseID)
		},
	)

	return router
}

// loggingMiddleware logs every dispatched message and any processing errors.
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
