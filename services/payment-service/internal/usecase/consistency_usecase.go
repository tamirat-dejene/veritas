package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"go.uber.org/zap"
)

type consistencyUseCase struct {
	subRepo     domain.SubscriptionRepository
	billingRepo domain.BillingRepository
	log         *zap.Logger
}

// NewConsistencyUseCase creates a ConsistencyUseCase that operates directly
// on the repository layer, bypassing normal usecase business guards.
func NewConsistencyUseCase(
	subRepo domain.SubscriptionRepository,
	billingRepo domain.BillingRepository,
	log *zap.Logger,
) domain.ConsistencyUseCase {
	return &consistencyUseCase{
		subRepo:     subRepo,
		billingRepo: billingRepo,
		log:         log.Named("consistency_usecase"),
	}
}

// HandleEnterpriseHardDeleted cancels the enterprise's subscription (if it
// exists and is not already canceled) and bulk-voids all outstanding open
// invoices. The operation is idempotent: if the subscription is already in a
// terminal state or does not exist, only the invoice voiding step is executed.
func (uc *consistencyUseCase) HandleEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID) error {
	uc.log.Info("handling enterprise hard-deleted event — cleaning up billing data",
		zap.String("enterprise_id", enterpriseID.String()),
	)

	// 1. Cancel the subscription record if it is still in a live state.
	sub, err := uc.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
	if err != nil && err != domain.ErrSubscriptionNotFound {
		uc.log.Error("failed to fetch subscription for hard-deleted enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Error(err),
		)
		return err
	}

	if sub != nil && sub.Status != domain.SubStatusCanceled && sub.Status != domain.SubStatusExpired {
		now := time.Now()
		sub.Status = domain.SubStatusCanceled
		sub.CancelAtPeriodEnd = false
		sub.CanceledAt = &now
		sub.EndedAt = &now
		sub.UpdatedAt = now

		if err := uc.subRepo.UpdateSubscription(ctx, sub); err != nil {
			uc.log.Error("failed to cancel subscription for hard-deleted enterprise",
				zap.String("enterprise_id", enterpriseID.String()),
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err),
			)
			return err
		}
		uc.log.Info("subscription canceled for hard-deleted enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.String("subscription_id", sub.ID.String()),
		)
	}

	// 2. Void all outstanding open invoices in a single bulk UPDATE.
	voided, err := uc.billingRepo.VoidOpenInvoices(ctx, enterpriseID)
	if err != nil {
		uc.log.Error("failed to void open invoices for hard-deleted enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Error(err),
		)
		return err
	}

	if voided > 0 {
		uc.log.Info("voided open invoices for hard-deleted enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Int64("voided_count", voided),
		)
	}

	return nil
}
