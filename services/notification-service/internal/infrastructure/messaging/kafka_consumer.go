package messaging

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/notification-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"go.uber.org/zap"
)

type NotificationRouter struct {
	uc     domain.NotificationUsecase
	logger *zap.Logger
}

func NewNotificationRouter(uc domain.NotificationUsecase, logger *zap.Logger) *NotificationRouter {
	return &NotificationRouter{
		uc:     uc,
		logger: logger,
	}
}

func (r *NotificationRouter) Topics() []string {
	return []string{
		topics.EnterpriseStaffCreated,
		topics.EnterprisePasswordResetRequested,
	}
}

func (r *NotificationRouter) Handle(ctx context.Context, msg messaging.Message) error {
	switch msg.Topic {
	case topics.EnterpriseStaffCreated:
		return r.uc.HandleEnterpriseStaffCreated(ctx, msg.Value)
	case topics.EnterprisePasswordResetRequested:
		return r.uc.HandlePasswordResetRequested(ctx, msg.Value)
	default:
		r.logger.Warn("Unhandled topic", zap.String("topic", msg.Topic))
		return nil
	}
}
