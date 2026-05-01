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
		topics.EnterpriseCreated,
		topics.EnterpriseApproved,
		topics.EnterpriseSuspended,
		topics.EnterpriseDeleted,
		topics.EnterpriseHardDeleted,
		topics.EnterpriseReactivated,
		topics.EnterpriseRestored,
		topics.UserDeactivated,
		topics.UserActivated,
		topics.UserPasswordChanged,
		topics.UserPasswordResetAdmin,
		topics.ExamCreated,
		topics.ExamScheduled,
		topics.ExamPublished,
		topics.ExamClosed,
		topics.CandidateEnrollmentInvited,
		topics.CandidateExamSubmitted,
	}
}

func (r *NotificationRouter) Handle(ctx context.Context, msg messaging.Message) error {
	switch msg.Topic {
	case topics.EnterpriseStaffCreated:
		return r.uc.HandleEnterpriseStaffCreated(ctx, msg.Value)
	case topics.EnterprisePasswordResetRequested:
		return r.uc.HandlePasswordResetRequested(ctx, msg.Value)
	case topics.EnterpriseCreated:
		return r.uc.HandleEnterpriseCreated(ctx, msg.Value)
	case topics.EnterpriseApproved:
		return r.uc.HandleEnterpriseApproved(ctx, msg.Value)
	case topics.EnterpriseSuspended:
		return r.uc.HandleEnterpriseSuspended(ctx, msg.Value)
	case topics.EnterpriseDeleted:
		return r.uc.HandleEnterpriseDeleted(ctx, msg.Value)
	case topics.EnterpriseHardDeleted:
		return r.uc.HandleEnterpriseHardDeleted(ctx, msg.Value)
	case topics.EnterpriseReactivated:
		return r.uc.HandleEnterpriseReactivated(ctx, msg.Value)
	case topics.EnterpriseRestored:
		return r.uc.HandleEnterpriseRestored(ctx, msg.Value)
	case topics.UserDeactivated:
		return r.uc.HandleUserDeactivated(ctx, msg.Value)
	case topics.UserActivated:
		return r.uc.HandleUserActivated(ctx, msg.Value)
	case topics.UserPasswordChanged:
		return r.uc.HandleUserPasswordChanged(ctx, msg.Value)
	case topics.UserPasswordResetAdmin:
		return r.uc.HandleUserPasswordResetAdmin(ctx, msg.Value)
	case topics.ExamCreated:
		return r.uc.HandleExamCreated(ctx, msg.Value)
	case topics.ExamScheduled:
		return r.uc.HandleExamScheduled(ctx, msg.Value)
	case topics.ExamPublished:
		return r.uc.HandleExamPublished(ctx, msg.Value)
	case topics.ExamClosed:
		return r.uc.HandleExamClosed(ctx, msg.Value)
	case topics.CandidateEnrollmentInvited:
		return r.uc.HandleCandidateEnrollmentInvited(ctx, msg.Value)
	case topics.CandidateExamSubmitted:
		return r.uc.HandleCandidateExamSubmitted(ctx, msg.Value)
	default:
		r.logger.Warn("Unhandled topic", zap.String("topic", msg.Topic))
		return nil
	}
}
