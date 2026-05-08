package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"go.uber.org/zap"
)

// paymentFailedEvent is the JSON payload published by payment-service on
// topics.SubscriptionPaymentFailed.
type paymentFailedEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Timestamp    int64     `json:"timestamp"`
}

// NewSubscriptionRouter creates a messaging.Router configured to handle
// all enterprise-service subscription events.
func NewSubscriptionRouter(usecase domain.EnterpriseUsecase, logger *zap.Logger) *messaging.Router {
	router := messaging.NewRouter(loggingMiddleware(logger))

	messaging.RegisterJSONHandler(router, topics.SubscriptionPaymentFailed, func(ctx context.Context, evt paymentFailedEvent) error {
		return usecase.SuspendForPayment(ctx, evt.EnterpriseID)
	})

	messaging.RegisterJSONHandler(router, topics.SubscriptionCanceled, func(ctx context.Context, evt paymentFailedEvent) error {
		return usecase.SuspendForPayment(ctx, evt.EnterpriseID)
	})

	return router
}

// loggingMiddleware provides a clean way to add cross-cutting concerns to all handlers.
func loggingMiddleware(logger *zap.Logger) messaging.Middleware {
	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg messaging.Message) error {
			logger.Debug("processing kafka message", 
				zap.String("topic", msg.Topic),
				zap.Int("payload_size", len(msg.Value)),
			)
			err := next(ctx, msg)
			if err != nil {
				logger.Error("failed to process kafka message",
					zap.String("topic", msg.Topic),
					zap.Error(err),
				)
			}
			return err
		}
	}
}
