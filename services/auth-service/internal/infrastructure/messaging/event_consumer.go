package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"go.uber.org/zap"
)

// userLifecycleEvent is the JSON payload published by enterprise-service for user events.
type userLifecycleEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	EnterpriseName string    `json:"enterprise_name"`
	TempPassword   string    `json:"temp_password,omitempty"`
	Timestamp      int64     `json:"timestamp"`
}

// enterpriseLifecycleEvent is the JSON payload published by enterprise-service for enterprise events.
type enterpriseLifecycleEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

// NewEventConsumer creates a messaging.Router configured to handle incoming consistency events.
func NewEventConsumer(usecase domain.ConsistencyUseCase, logger *zap.Logger) *messaging.Router {
	router := messaging.NewRouter(loggingMiddleware(logger))

	// User-level events
	userHandler := func(ctx context.Context, evt userLifecycleEvent) error {
		return usecase.RevokeUserSessions(ctx, evt.UserID)
	}

	messaging.RegisterJSONHandler(router, topics.UserDeleted, userHandler)
	messaging.RegisterJSONHandler(router, topics.UserDeactivated, userHandler)
	messaging.RegisterJSONHandler(router, topics.UserPasswordChanged, userHandler)
	messaging.RegisterJSONHandler(router, topics.UserPasswordResetAdmin, userHandler)

	// Enterprise-level events
	enterpriseHandler := func(ctx context.Context, evt enterpriseLifecycleEvent) error {
		return usecase.RevokeEnterpriseSessions(ctx, evt.EnterpriseID)
	}

	messaging.RegisterJSONHandler(router, topics.EnterpriseDeleted, enterpriseHandler)
	messaging.RegisterJSONHandler(router, topics.EnterpriseSuspended, enterpriseHandler)
	messaging.RegisterJSONHandler(router, topics.EnterpriseHardDeleted, enterpriseHandler)

	return router
}

// loggingMiddleware provides a clean way to add cross-cutting concerns to all handlers.
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
