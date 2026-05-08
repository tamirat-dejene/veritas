package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"go.uber.org/zap"
)

// enterpriseLifecycleEvent is the JSON payload published by enterprise-service.
type enterpriseLifecycleEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

// examLifecycleEvent is the JSON payload published by exam-service for exam events.
type examLifecycleEvent struct {
	ExamID       uuid.UUID `json:"exam_id"`
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	Timestamp    int64     `json:"timestamp"`
}

// NewEventConsumer creates a messaging.Router configured to handle incoming consistency events.
func NewEventConsumer(usecase domain.ConsistencyUseCase, logger *zap.Logger) *messaging.Router {
	router := messaging.NewRouter(loggingMiddleware(logger))

	// Enterprise-level events
	enterpriseHandler := func(ctx context.Context, evt enterpriseLifecycleEvent) error {
		return usecase.HandleEnterpriseDeactivated(ctx, evt.EnterpriseID)
	}

	messaging.RegisterJSONHandler(router, topics.EnterpriseSuspended, enterpriseHandler)
	messaging.RegisterJSONHandler(router, topics.EnterpriseDeleted, enterpriseHandler)
	messaging.RegisterJSONHandler(router, topics.EnterpriseHardDeleted, enterpriseHandler)

	// Exam-level events
	examHandler := func(ctx context.Context, evt examLifecycleEvent) error {
		return usecase.HandleExamClosed(ctx, evt.ExamID)
	}

	messaging.RegisterJSONHandler(router, topics.ExamClosed, examHandler)

	return router
}

func loggingMiddleware(logger *zap.Logger) messaging.Middleware {
	return func(next messaging.Handler) messaging.Handler {
		return func(ctx context.Context, msg messaging.Message) error {
			logger.Debug("processing kafka message for consistency", 
				zap.String("topic", msg.Topic),
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
