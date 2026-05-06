package usecase

import (
	"context"
	"time"

	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"go.uber.org/zap"
)

type maintenanceUseCase struct {
	subRepo        domain.SubscriptionRepository
	billingRepo    domain.BillingRepository
	eventPublisher domain.PaymentEventPublisher
	log            *zap.Logger
}

func NewMaintenanceUseCase(
	subRepo domain.SubscriptionRepository,
	billingRepo domain.BillingRepository,
	eventPublisher domain.PaymentEventPublisher,
	log *zap.Logger,
) domain.MaintenanceUseCase {
	return &maintenanceUseCase{
		subRepo:        subRepo,
		billingRepo:    billingRepo,
		eventPublisher: eventPublisher,
		log:            log.Named("maintenance_usecase"),
	}
}

func (uc *maintenanceUseCase) ExpireLapsedSubscriptions(ctx context.Context) error {
	batchSize := 100

	// 1. Expire subscriptions that are cancel-at-period-end and past their period
	lapsed, err := uc.subRepo.GetLapsedSubscriptions(ctx, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch lapsed subscriptions", zap.Error(err))
		return err
	}

	if len(lapsed) > 0 {
		uc.log.Info("expiring lapsed subscriptions", zap.Int("count", len(lapsed)))
	}

	for _, sub := range lapsed {
		now := time.Now()
		sub.Status = domain.SubStatusExpired
		sub.EndedAt = &now
		sub.UpdatedAt = now

		if err := uc.subRepo.UpdateSubscription(ctx, sub); err != nil {
			uc.log.Error("failed to expire subscription",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err))
			continue
		}

		if uc.eventPublisher != nil {
			if err := uc.eventPublisher.PublishSubscriptionCanceled(ctx, sub.EnterpriseID); err != nil {
				uc.log.Error("failed to publish subscription canceled event",
					zap.String("enterprise_id", sub.EnterpriseID.String()),
					zap.Error(err))
			}
		}
	}

	// 2. Mark past-due candidates (not cancel-at-period-end but period has ended)
	pastDue, err := uc.subRepo.GetPastDueCandidates(ctx, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch past-due candidates", zap.Error(err))
		return err
	}

	if len(pastDue) > 0 {
		uc.log.Info("marking subscriptions as past due", zap.Int("count", len(pastDue)))
	}

	for _, sub := range pastDue {
		sub.Status = domain.SubStatusPastDue
		sub.UpdatedAt = time.Now()

		if err := uc.subRepo.UpdateSubscription(ctx, sub); err != nil {
			uc.log.Error("failed to mark subscription as past due",
				zap.String("subscription_id", sub.ID.String()),
				zap.Error(err))
			continue
		}

		if uc.eventPublisher != nil {
			if err := uc.eventPublisher.PublishPaymentFailed(ctx, sub.EnterpriseID); err != nil {
				uc.log.Error("failed to publish payment failed event",
					zap.String("enterprise_id", sub.EnterpriseID.String()),
					zap.Error(err))
			}
		}
	}

	return nil
}

func (uc *maintenanceUseCase) MarkOverdueInvoicesUncollectible(ctx context.Context) error {
	graceDays := 14
	batchSize := 100

	invoices, err := uc.billingRepo.GetOverdueInvoices(ctx, graceDays, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch overdue invoices", zap.Error(err))
		return err
	}

	if len(invoices) == 0 {
		return nil
	}

	uc.log.Info("marking overdue invoices as uncollectible", zap.Int("count", len(invoices)))

	successCount := 0
	for _, inv := range invoices {
		inv.Status = domain.InvoiceStatusUncollectible
		inv.UpdatedAt = time.Now()

		if err := uc.billingRepo.UpdateInvoice(ctx, inv); err != nil {
			uc.log.Error("failed to mark invoice as uncollectible",
				zap.String("invoice_id", inv.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	uc.log.Info("completed marking overdue invoices",
		zap.Int("total", len(invoices)),
		zap.Int("success", successCount))

	return nil
}

func (uc *maintenanceUseCase) PurgeOldWebhookEvents(ctx context.Context) error {
	retentionDays := 30
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	count, err := uc.billingRepo.PurgeOldWebhookEvents(ctx, cutoff)
	if err != nil {
		uc.log.Error("failed to purge old webhook events", zap.Error(err))
		return err
	}
	if count > 0 {
		uc.log.Info("purged old webhook events", zap.Int64("count", count), zap.Time("cutoff", cutoff))
	}
	return nil
}
