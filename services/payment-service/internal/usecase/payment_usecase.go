package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v74"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"go.uber.org/zap"
)

type paymentUsecase struct {
	pool           *pgxpool.Pool
	subRepo        domain.SubscriptionRepository
	billingRepo    domain.BillingRepository
	payProvider    domain.PaymentProvider
	eventPublisher domain.PaymentEventPublisher
}

func NewPaymentUsecase(
	pool *pgxpool.Pool,
	subRepo domain.SubscriptionRepository,
	billingRepo domain.BillingRepository,
	payProvider domain.PaymentProvider,
	eventPublisher domain.PaymentEventPublisher,
) domain.PaymentUsecase {
	return &paymentUsecase{
		pool:           pool,
		subRepo:        subRepo,
		billingRepo:    billingRepo,
		payProvider:    payProvider,
		eventPublisher: eventPublisher,
	}
}

// ─── Plans ────────────────────────────────────────────────────────────────────

func (u *paymentUsecase) ListPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	return u.subRepo.ListPlans(ctx)
}

// ─── Subscriptions ────────────────────────────────────────────────────────────

func (u *paymentUsecase) GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*domain.EnterpriseSubscription, error) {
	return u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
}

func (u *paymentUsecase) UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID) (string, error) {
	plan, err := u.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return "", err
	}
	return u.payProvider.CreateCheckoutSession(ctx, enterpriseID, plan)
}

// CancelSubscription cancels an enterprise's Stripe subscription.
// When cancelAtPeriodEnd is true, the subscription remains active until the
// billing period ends. When false, it is canceled immediately.
func (u *paymentUsecase) CancelSubscription(ctx context.Context, enterpriseID uuid.UUID, cancelAtPeriodEnd bool) error {
	sub, err := u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if sub.Status == domain.SubStatusCanceled {
		return domain.ErrSubscriptionAlreadyCanceled
	}
	if sub.StripeSubscriptionID == nil {
		return fmt.Errorf("subscription has no associated Stripe subscription ID")
	}

	// Cancel via Stripe
	if err := u.payProvider.CancelStripeSubscription(ctx, *sub.StripeSubscriptionID, cancelAtPeriodEnd); err != nil {
		return fmt.Errorf("cancel stripe subscription: %w", err)
	}

	// Update local record
	now := time.Now()
	sub.CancelAtPeriodEnd = cancelAtPeriodEnd
	if !cancelAtPeriodEnd {
		sub.Status = domain.SubStatusCanceled
		sub.CanceledAt = &now
		sub.EndedAt = &now
	}
	sub.UpdatedAt = now
	return u.subRepo.UpdateSubscription(ctx, sub)
}

// ReactivateSubscription un-schedules a pending cancellation on a subscription.
func (u *paymentUsecase) ReactivateSubscription(ctx context.Context, enterpriseID uuid.UUID) error {
	sub, err := u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if sub.Status == domain.SubStatusCanceled {
		return domain.ErrSubscriptionAlreadyCanceled
	}
	if !sub.CancelAtPeriodEnd {
		return fmt.Errorf("subscription is not scheduled for cancellation")
	}
	if sub.StripeSubscriptionID == nil {
		return fmt.Errorf("subscription has no associated Stripe subscription ID")
	}

	if err := u.payProvider.ReactivateStripeSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
		return fmt.Errorf("reactivate stripe subscription: %w", err)
	}

	sub.CancelAtPeriodEnd = false
	sub.UpdatedAt = time.Now()
	return u.subRepo.UpdateSubscription(ctx, sub)
}

// AdminSetSubscription lets a system admin manually override subscription state
// (e.g. for trials, manual plans). No Stripe call is made.
func (u *paymentUsecase) AdminSetSubscription(ctx context.Context, enterpriseID uuid.UUID, req domain.AdminSetSubscriptionRequest) error {
	now := time.Now()

	periodStart := now
	if req.PeriodStart != nil {
		periodStart = *req.PeriodStart
	}
	periodEnd := u.calculatePeriodEnd(periodStart, domain.BillingCycleMonthly)
	if req.PeriodEnd != nil {
		periodEnd = *req.PeriodEnd
	}

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		existing, err := u.subRepo.WithTx(tx).GetSubscriptionByEnterpriseID(ctx, enterpriseID)
		if err != nil && err != domain.ErrSubscriptionNotFound {
			return err
		}

		if err == domain.ErrSubscriptionNotFound || existing == nil {
			// Create new subscription record
			newSub := &domain.EnterpriseSubscription{
				ID:                 uuid.New(),
				EnterpriseID:       enterpriseID,
				PlanID:             req.PlanID,
				Status:             req.Status,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
		}

		// Update existing
		existing.PlanID = req.PlanID
		existing.Status = req.Status
		existing.CurrentPeriodStart = periodStart
		existing.CurrentPeriodEnd = periodEnd
		existing.UpdatedAt = now
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, existing)
	})
}

// ─── Webhook ──────────────────────────────────────────────────────────────────

func (u *paymentUsecase) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	eventAny, err := u.payProvider.ConstructEvent(payload, sigHeader)
	if err != nil {
		return err
	}

	event, ok := eventAny.(*stripe.Event)
	if !ok {
		return fmt.Errorf("unexpected event type")
	}

	// Idempotency check
	processed, err := u.billingRepo.HasEventBeenProcessed(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("check event processed: %w", err)
	}
	if processed {
		return nil // Already processed, return success
	}

	var processErr error
	switch event.Type {
	case "checkout.session.completed":
		processErr = u.handleCheckoutSessionCompleted(ctx, event)
	case "invoice.paid":
		processErr = u.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		processErr = u.handleInvoicePaymentFailed(ctx, event)
	case "customer.subscription.updated":
		processErr = u.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		processErr = u.handleSubscriptionDeleted(ctx, event)
	}

	if processErr != nil {
		return processErr
	}

	// Record success
	return u.billingRepo.RecordEventProcessed(ctx, event.ID, string(event.Type))
}

func (u *paymentUsecase) handleCheckoutSessionCompleted(ctx context.Context, event *stripe.Event) error {
	metadata := event.Data.Object["metadata"].(map[string]any)
	enterpriseIDStr := metadata["enterprise_id"].(string)
	planIDStr := metadata["plan_id"].(string)

	customerStr, _ := event.Data.Object["customer"].(string)
	subscriptionStr, _ := event.Data.Object["subscription"].(string)

	enterpriseID, _ := uuid.Parse(enterpriseIDStr)
	planID, _ := uuid.Parse(planIDStr)

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		plan, err := u.subRepo.WithTx(tx).GetPlanByID(ctx, planID)
		if err != nil {
			return err
		}

		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByEnterpriseID(ctx, enterpriseID)
		if err != nil {
			if err == domain.ErrSubscriptionNotFound {
				newSub := &domain.EnterpriseSubscription{
					ID:                 uuid.New(),
					EnterpriseID:       enterpriseID,
					PlanID:             planID,
					Status:             domain.SubStatusActive,
					CurrentPeriodStart: time.Now(),
					CurrentPeriodEnd:   u.calculatePeriodEnd(time.Now(), plan.BillingCycle),
				}
				if customerStr != "" {
					newSub.StripeCustomerID = &customerStr
				}
				if subscriptionStr != "" {
					newSub.StripeSubscriptionID = &subscriptionStr
				}
				return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
			}
			return err
		}

		sub.PlanID = planID
		sub.Status = domain.SubStatusActive
		sub.UpdatedAt = time.Now()
		if customerStr != "" {
			sub.StripeCustomerID = &customerStr
		}
		if subscriptionStr != "" {
			sub.StripeSubscriptionID = &subscriptionStr
		}
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub)
	})
}

func (u *paymentUsecase) handleInvoicePaid(ctx context.Context, event *stripe.Event) error {
	invoiceObj := event.Data.Object
	invoiceNumber := invoiceObj["number"].(string)
	amountPaidRaw := invoiceObj["amount_paid"]
	var amountPaid float64
	switch v := amountPaidRaw.(type) {
	case float64:
		amountPaid = v / 100
	case int64:
		amountPaid = float64(v) / 100
	}
	currency := domain.Currency(invoiceObj["currency"].(string))

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		inv, err := u.billingRepo.WithTx(tx).GetInvoiceByNumber(ctx, invoiceNumber)
		if err != nil {
			return err
		}

		inv.Status = domain.InvoiceStatusPaid
		inv.AmountPaid = amountPaid
		inv.AmountRemaining = 0
		now := time.Now()
		inv.PaidAt = &now
		if err := u.billingRepo.WithTx(tx).UpdateInvoice(ctx, inv); err != nil {
			return err
		}

		payment := &domain.Payment{
			ID:                uuid.New(),
			EnterpriseID:      inv.EnterpriseID,
			InvoiceID:         &inv.ID,
			Amount:            amountPaid,
			Currency:          currency,
			Status:            domain.PaymentStatusSucceeded,
			Provider:          "stripe",
			ProviderPaymentID: event.ID,
			CreatedAt:         time.Now(),
		}
		return u.billingRepo.WithTx(tx).CreatePayment(ctx, payment)
	})
}

// handleInvoicePaymentFailed is called when Stripe fires invoice.payment_failed.
// It marks the invoice as Uncollectible, updates the subscription to PastDue,
// and publishes a Kafka event so enterprise-service can suspend the enterprise.
func (u *paymentUsecase) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	invoiceObj := event.Data.Object
	invoiceNumber, _ := invoiceObj["number"].(string)

	// Extract the enterprise_id from subscription metadata if possible
	// Fall back to looking up by invoice number
	var enterpriseID uuid.UUID

	if err := RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		if invoiceNumber == "" {
			return nil
		}
		inv, err := u.billingRepo.WithTx(tx).GetInvoiceByNumber(ctx, invoiceNumber)
		if err != nil {
			return err
		}
		enterpriseID = inv.EnterpriseID

		// Mark invoice as Uncollectible
		inv.Status = domain.InvoiceStatusUncollectible
		if err := u.billingRepo.WithTx(tx).UpdateInvoice(ctx, inv); err != nil {
			return err
		}

		// Mark subscription as PastDue
		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByEnterpriseID(ctx, inv.EnterpriseID)
		if err != nil {
			return err
		}
		sub.Status = domain.SubStatusPastDue
		sub.UpdatedAt = time.Now()
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub)
	}); err != nil {
		return err
	}

	// Publish Kafka event — non-fatal: log and continue if Kafka is unavailable.
	if enterpriseID != uuid.Nil {
		if err := u.eventPublisher.PublishPaymentFailed(ctx, enterpriseID); err != nil {
			zap.L().Error("handleInvoicePaymentFailed: failed to publish kafka event",
				zap.String("enterprise_id", enterpriseID.String()),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (u *paymentUsecase) handleSubscriptionUpdated(ctx context.Context, event *stripe.Event) error {
	subObj := event.Data.Object
	stripeSubscriptionID, _ := subObj["id"].(string)
	if stripeSubscriptionID == "" {
		return nil
	}

	status, _ := subObj["status"].(string)
	currentPeriodStartRaw, _ := subObj["current_period_start"].(float64)
	currentPeriodEndRaw, _ := subObj["current_period_end"].(float64)
	cancelAtPeriodEnd, _ := subObj["cancel_at_period_end"].(bool)

	var subStatus domain.SubscriptionStatus
	switch status {
	case "active":
		subStatus = domain.SubStatusActive
	case "past_due":
		subStatus = domain.SubStatusPastDue
	case "canceled":
		subStatus = domain.SubStatusCanceled
	case "trialing":
		subStatus = domain.SubStatusTrial
	default:
		subStatus = domain.SubStatusExpired
	}

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByStripeID(ctx, stripeSubscriptionID)
		if err != nil {
			if err == domain.ErrSubscriptionNotFound {
				return nil // Ignore if we don't have it locally
			}
			return err
		}

		sub.Status = subStatus
		sub.CurrentPeriodStart = time.Unix(int64(currentPeriodStartRaw), 0)
		sub.CurrentPeriodEnd = time.Unix(int64(currentPeriodEndRaw), 0)
		sub.CancelAtPeriodEnd = cancelAtPeriodEnd
		sub.UpdatedAt = time.Now()

		if err := u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub); err != nil {
			return err
		}

		if err := u.eventPublisher.PublishSubscriptionUpdated(ctx, sub.EnterpriseID); err != nil {
			zap.L().Error("failed to publish subscription_updated event", zap.Error(err))
		}

		return nil
	})
}

func (u *paymentUsecase) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) error {
	subObj := event.Data.Object
	stripeSubscriptionID, _ := subObj["id"].(string)
	if stripeSubscriptionID == "" {
		return nil
	}

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByStripeID(ctx, stripeSubscriptionID)
		if err != nil {
			if err == domain.ErrSubscriptionNotFound {
				return nil
			}
			return err
		}

		now := time.Now()
		sub.Status = domain.SubStatusCanceled
		sub.CanceledAt = &now
		sub.EndedAt = &now
		sub.UpdatedAt = now
		sub.CancelAtPeriodEnd = false

		if err := u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub); err != nil {
			return err
		}

		if err := u.eventPublisher.PublishSubscriptionCanceled(ctx, sub.EnterpriseID); err != nil {
			zap.L().Error("failed to publish subscription_canceled event", zap.Error(err))
		}

		return nil
	})
}

// ─── Invoices & Payments ──────────────────────────────────────────────────────

func (u *paymentUsecase) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error) {
	return u.billingRepo.GetInvoiceByID(ctx, invoiceID)
}

func (u *paymentUsecase) ListInvoices(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Invoice, error) {
	return u.billingRepo.ListInvoicesByEnterprise(ctx, enterpriseID)
}

func (u *paymentUsecase) ListPaymentHistory(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Payment, error) {
	return u.billingRepo.ListPaymentsByEnterprise(ctx, enterpriseID)
}

func (u *paymentUsecase) calculatePeriodEnd(start time.Time, cycle domain.BillingCycle) time.Time {
	if cycle == domain.BillingCycleYearly {
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 1, 0)
}
