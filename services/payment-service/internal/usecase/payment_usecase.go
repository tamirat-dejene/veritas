package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/infrastructure/providerregistry"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
)

type paymentUsecase struct {
	pool             *pgxpool.Pool
	subRepo          domain.SubscriptionRepository
	billingRepo      domain.BillingRepository
	providerRegistry *providerregistry.ProviderRegistry
	eventPublisher   domain.PaymentEventPublisher
}

func NewPaymentUsecase(
	pool *pgxpool.Pool,
	subRepo domain.SubscriptionRepository,
	billingRepo domain.BillingRepository,
	providerRegistry *providerregistry.ProviderRegistry,
	eventPublisher domain.PaymentEventPublisher,
) domain.PaymentUsecase {
	return &paymentUsecase{
		pool:             pool,
		subRepo:          subRepo,
		billingRepo:      billingRepo,
		providerRegistry: providerRegistry,
		eventPublisher:   eventPublisher,
	}
}

// ─── Plans ────────────────────────────────────────────────────────────────────

func (u *paymentUsecase) ListPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*domain.SubscriptionPlan], error) {
	plans, total, err := u.subRepo.ListPlans(ctx, params)
	if err != nil {
		return pagination.PaginatedResponse[*domain.SubscriptionPlan]{}, err
	}
	return pagination.NewPaginatedResponse(plans, total, params), nil
}

func (u *paymentUsecase) ListAllPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*domain.SubscriptionPlan], error) {
	plans, total, err := u.subRepo.ListAllPlans(ctx, params)
	if err != nil {
		return pagination.PaginatedResponse[*domain.SubscriptionPlan]{}, err
	}
	return pagination.NewPaginatedResponse(plans, total, params), nil
}

func (u *paymentUsecase) GetPlanByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	return u.subRepo.GetPlanByID(ctx, id)
}

func (u *paymentUsecase) CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	if plan.StripePriceID == "" {
		stripeProv, err := u.providerRegistry.Get(domain.PaymentProviderStripe)
		if err == nil {
			stripeID, err := stripeProv.SyncPlan(ctx, plan)
			if err != nil {
				return fmt.Errorf("sync plan to stripe: %w", err)
			}
			plan.StripePriceID = stripeID
		} else {
			zap.L().Warn("stripe provider not found in registry during CreatePlan", zap.Error(err))
		}
	}
	return u.subRepo.CreatePlan(ctx, plan)
}

func (u *paymentUsecase) UpdatePlan(ctx context.Context, plan *domain.SubscriptionPlan) error {
	return u.subRepo.UpdatePlan(ctx, plan)
}

func (u *paymentUsecase) DeactivatePlan(ctx context.Context, planID uuid.UUID) error {
	plan, err := u.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return err
	}
	plan.IsActive = false

	if plan.StripePriceID != "" {
		stripeProv, err := u.providerRegistry.Get(domain.PaymentProviderStripe)
		if err == nil {
			if err := stripeProv.DeactivatePlan(ctx, plan.StripePriceID); err != nil {
				zap.L().Warn("failed to deactivate stripe price during plan deactivation",
					zap.String("plan_id", planID.String()),
					zap.String("stripe_price_id", plan.StripePriceID),
					zap.Error(err),
				)
			}
		}
	}

	return u.subRepo.UpdatePlan(ctx, plan)
}

// ─── Subscriptions ────────────────────────────────────────────────────────────

func (u *paymentUsecase) GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*domain.EnterpriseSubscription, error) {
	return u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
}

func (u *paymentUsecase) UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID, provider string) (string, error) {
	if provider == "" {
		provider = domain.PaymentProviderStripe
	}

	plan, err := u.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return "", err
	}

	if provider == domain.PaymentProviderChapa {
		if plan.Currency != domain.CurrencyETB {
			return "", fmt.Errorf("chapa: %w: only ETB plans are supported, got %s", domain.ErrInvalidInput, plan.Currency)
		}
	}

	prov, err := u.providerRegistry.Get(provider)
	if err != nil {
		return "", err
	}

	var stripeCustomerID *string
	sub, err := u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
	if err == nil && sub != nil {
		stripeCustomerID = sub.StripeCustomerID
	}

	txRef := ""
	if provider == domain.PaymentProviderChapa {
		txRef = fmt.Sprintf("veritas-%s-%d", enterpriseID.String()[:8], time.Now().Unix())
	}

	req := domain.CheckoutRequest{
		EnterpriseID: enterpriseID,
		Plan:         plan,
		CustomerRef:  stripeCustomerID,
		TxRef:        txRef,
	}

	checkoutURL, err := prov.CreateCheckoutSession(ctx, req)
	if err != nil {
		return "", err
	}

	// Update local subscription to store the tx_ref and provider if it's Chapa
	if provider == domain.PaymentProviderChapa {
		if err := RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
			existing, err := u.subRepo.WithTx(tx).GetSubscriptionByEnterpriseID(ctx, enterpriseID)
			if err != nil && err != domain.ErrSubscriptionNotFound {
				return err
			}

			if err == domain.ErrSubscriptionNotFound || existing == nil {
				newSub := &domain.EnterpriseSubscription{
					ID:                 uuid.New(),
					EnterpriseID:       enterpriseID,
					PlanID:             planID,
					Status:             domain.SubStatusExpired, // Starts as expired/pending
					CurrentPeriodStart: time.Now(),
					CurrentPeriodEnd:   time.Now(),
					ChapaTxRef:         &txRef,
					PaymentProvider:    domain.PaymentProviderChapa,
				}
				return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
			}

			existing.ChapaTxRef = &txRef
			existing.PaymentProvider = domain.PaymentProviderChapa
			return u.subRepo.WithTx(tx).UpdateSubscription(ctx, existing)
		}); err != nil {
			return "", fmt.Errorf("save chapa tx ref: %w", err)
		}
	}

	return checkoutURL, nil
}

// CancelSubscription cancels an enterprise's subscription.
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

	// Default to stripe if provider not specified
	provider := sub.PaymentProvider
	if provider == "" {
		provider = domain.PaymentProviderStripe
	}

	if provider == domain.PaymentProviderStripe {
		if sub.StripeSubscriptionID == nil {
			return fmt.Errorf("subscription has no associated Stripe subscription ID")
		}
		prov, err := u.providerRegistry.Get(domain.PaymentProviderStripe)
		if err != nil {
			return err
		}
		if err := prov.CancelSubscription(ctx, *sub.StripeSubscriptionID, cancelAtPeriodEnd); err != nil {
			return fmt.Errorf("cancel stripe subscription: %w", err)
		}
	} else if provider == domain.PaymentProviderChapa {
		// Chapa subscriptions don't have a recurring auto-billing contract to cancel.
		// Cancellation will take effect locally.
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

	provider := sub.PaymentProvider
	if provider == "" {
		provider = domain.PaymentProviderStripe
	}

	if provider == domain.PaymentProviderStripe {
		if sub.StripeSubscriptionID == nil {
			return fmt.Errorf("subscription has no associated Stripe subscription ID")
		}
		prov, err := u.providerRegistry.Get(domain.PaymentProviderStripe)
		if err != nil {
			return err
		}
		if err := prov.ReactivateSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
			return fmt.Errorf("reactivate stripe subscription: %w", err)
		}
	}

	sub.CancelAtPeriodEnd = false
	sub.UpdatedAt = time.Now()
	return u.subRepo.UpdateSubscription(ctx, sub)
}

// AdminSetSubscription lets a system admin manually override subscription state
// (e.g. for trials, manual plans). No payment provider call is made.
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
				PaymentProvider:    domain.PaymentProviderStripe, // default to stripe
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
		}

		// Apply admin changes to existing subscription
		existing.PlanID = req.PlanID
		existing.Status = req.Status
		existing.CurrentPeriodStart = periodStart
		existing.CurrentPeriodEnd = periodEnd
		existing.UpdatedAt = now
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, existing)
	})
}

// CreateTrialSubscription provisions a free trial for the enterprise without requiring any payment provider.
func (u *paymentUsecase) CreateTrialSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID, trialDays int) error {
	now := time.Now()
	periodEnd := now.AddDate(0, 0, trialDays)

	err := RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		existing, err := u.subRepo.WithTx(tx).GetSubscriptionByEnterpriseID(ctx, enterpriseID)
		if err != nil && err != domain.ErrSubscriptionNotFound {
			return err
		}

		if err == domain.ErrSubscriptionNotFound || existing == nil {
			newSub := &domain.EnterpriseSubscription{
				ID:                 uuid.New(),
				EnterpriseID:       enterpriseID,
				PlanID:             planID,
				Status:             domain.SubStatusTrial,
				CurrentPeriodStart: now,
				CurrentPeriodEnd:   periodEnd,
				PaymentProvider:    domain.PaymentProviderStripe,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
		}

		// Update existing
		existing.PlanID = planID
		existing.Status = domain.SubStatusTrial
		existing.CurrentPeriodStart = now
		existing.CurrentPeriodEnd = periodEnd
		existing.UpdatedAt = now
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, existing)
	})

	if err != nil {
		return err
	}

	// Publish event so enterprise-service knows the subscription changed
	if u.eventPublisher != nil {
		return u.eventPublisher.PublishSubscriptionUpdated(ctx, enterpriseID)
	}
	return nil
}

// ─── Webhook ──────────────────────────────────────────────────────────────────

func (u *paymentUsecase) HandleWebhook(ctx context.Context, payload []byte, sigHeader string, provider string) error {
	if provider == "" {
		provider = domain.PaymentProviderStripe
	}

	prov, err := u.providerRegistry.Get(provider)
	if err != nil {
		return err
	}

	event, err := prov.VerifyWebhookEvent(payload, sigHeader)
	if err != nil {
		return err
	}

	// Idempotency check
	processed, err := u.billingRepo.HasEventBeenProcessed(ctx, event.EventID)
	if err != nil {
		return fmt.Errorf("check event processed: %w", err)
	}
	if processed {
		return nil // Already processed, return success
	}

	var processErr error
	if provider == domain.PaymentProviderChapa {
		if event.EventType == "payment.success" {
			processErr = u.handleChapaPaymentSuccess(ctx, event)
		} else if event.EventType == "payment.failed" {
			processErr = u.handleChapaPaymentFailed(ctx, event)
		}
	} else {
		// Stripe specific events
		switch event.EventType {
		case "checkout.session.completed":
			processErr = u.handleCheckoutSessionCompleted(ctx, event)
		case "invoice.paid":
			processErr = u.handleInvoicePaid(ctx, event)
		case "invoice.payment_failed":
			processErr = u.handleInvoicePaymentFailed(ctx, event)
		case "invoice.upcoming":
			processErr = u.handleInvoiceUpcoming(ctx, event)
		case "customer.subscription.updated":
			processErr = u.handleSubscriptionUpdated(ctx, event)
		case "customer.subscription.deleted":
			processErr = u.handleSubscriptionDeleted(ctx, event)
		}
	}

	if processErr != nil {
		return processErr
	}

	// Record success
	return u.billingRepo.RecordEventProcessed(ctx, event.EventID, string(event.EventType))
}

func (u *paymentUsecase) handleChapaPaymentSuccess(ctx context.Context, event *domain.PaymentEvent) error {
	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		// 1. Find subscription by ChapaTxRef
		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByChapaTxRef(ctx, event.TxRef)
		if err != nil {
			return fmt.Errorf("find subscription by tx_ref %s: %w", event.TxRef, err)
		}

		plan, err := u.subRepo.WithTx(tx).GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return fmt.Errorf("get plan by ID %s: %w", sub.PlanID, err)
		}

		// Update period
		now := time.Now()
		sub.Status = domain.SubStatusActive
		sub.CurrentPeriodStart = now
		sub.CurrentPeriodEnd = u.calculatePeriodEnd(now, plan.BillingCycle)
		sub.UpdatedAt = now

		if err := u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub); err != nil {
			return fmt.Errorf("update subscription: %w", err)
		}

		// 2. Generate a local paid invoice for Chapa payment
		invNumber := fmt.Sprintf("INV-%d", now.UnixNano()/1e6) // Unique millisecond-based invoice number
		invoice := &domain.Invoice{
			ID:              uuid.New(),
			EnterpriseID:    sub.EnterpriseID,
			SubscriptionID:  sub.ID,
			Number:          invNumber,
			Status:          domain.InvoiceStatusPaid,
			AmountDue:       plan.Price,
			AmountPaid:      plan.Price,
			AmountRemaining: 0,
			Currency:        plan.Currency,
			DueDate:         now,
			PaidAt:          &now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := u.billingRepo.WithTx(tx).CreateInvoice(ctx, invoice); err != nil {
			return fmt.Errorf("create invoice: %w", err)
		}

		// 3. Create a payment record
		payment := &domain.Payment{
			ID:                uuid.New(),
			EnterpriseID:      sub.EnterpriseID,
			InvoiceID:         &invoice.ID,
			Amount:            event.Amount,
			Currency:          event.Currency,
			Status:            domain.PaymentStatusSucceeded,
			Provider:          domain.PaymentProviderChapa,
			ProviderPaymentID: event.TxRef, // Using tx_ref as provider_payment_id
			CreatedAt:         now,
		}

		if err := u.billingRepo.WithTx(tx).CreatePayment(ctx, payment); err != nil {
			return fmt.Errorf("create payment record: %w", err)
		}

		// Publish event
		if u.eventPublisher != nil {
			if pubErr := u.eventPublisher.PublishSubscriptionUpdated(ctx, sub.EnterpriseID); pubErr != nil {
				zap.L().Error("failed to publish subscription_updated event after Chapa checkout", zap.Error(pubErr))
			}
		}

		return nil
	})
}

func (u *paymentUsecase) handleChapaPaymentFailed(ctx context.Context, event *domain.PaymentEvent) error {
	var enterpriseID uuid.UUID
	if err := RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		sub, err := u.subRepo.WithTx(tx).GetSubscriptionByChapaTxRef(ctx, event.TxRef)
		if err != nil {
			return err
		}
		enterpriseID = sub.EnterpriseID

		sub.Status = domain.SubStatusPastDue
		sub.UpdatedAt = time.Now()
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub)
	}); err != nil {
		return err
	}

	if enterpriseID != uuid.Nil && u.eventPublisher != nil {
		if err := u.eventPublisher.PublishPaymentFailed(ctx, enterpriseID); err != nil {
			zap.L().Error("handleChapaPaymentFailed: failed to publish kafka event",
				zap.String("enterprise_id", enterpriseID.String()),
				zap.Error(err),
			)
		}
	}
	return nil
}

func (u *paymentUsecase) handleCheckoutSessionCompleted(ctx context.Context, event *domain.PaymentEvent) error {
	metadata, _ := event.Raw["metadata"].(map[string]any)
	enterpriseIDStr, _ := metadata["enterprise_id"].(string)
	planIDStr, _ := metadata["plan_id"].(string)

	customerStr := event.CustomerRef
	subscriptionStr := event.SubscriptionRef

	enterpriseID, _ := uuid.Parse(enterpriseIDStr)
	planID, _ := uuid.Parse(planIDStr)

	err := RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
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
					PaymentProvider:    domain.PaymentProviderStripe,
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
		sub.PaymentProvider = domain.PaymentProviderStripe
		sub.UpdatedAt = time.Now()
		if customerStr != "" {
			sub.StripeCustomerID = &customerStr
		}
		if subscriptionStr != "" {
			sub.StripeSubscriptionID = &subscriptionStr
		}
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub)
	})
	if err == nil && u.eventPublisher != nil {
		if pubErr := u.eventPublisher.PublishSubscriptionUpdated(ctx, enterpriseID); pubErr != nil {
			zap.L().Error("failed to publish subscription_updated event after checkout", zap.Error(pubErr))
		}
	}
	return err
}

func (u *paymentUsecase) handleInvoicePaid(ctx context.Context, event *domain.PaymentEvent) error {
	invoiceNumber, _ := event.Raw["number"].(string)
	amountPaidRaw := event.Raw["amount_paid"]
	var amountPaid float64
	switch v := amountPaidRaw.(type) {
	case float64:
		amountPaid = v / 100
	case int64:
		amountPaid = float64(v) / 100
	}
	amountDueRaw := event.Raw["amount_due"]
	var amountDue float64
	switch v := amountDueRaw.(type) {
	case float64:
		amountDue = v / 100
	case int64:
		amountDue = float64(v) / 100
	}
	currencyStr, _ := event.Raw["currency"].(string)
	currency := domain.Currency(currencyStr)
	hostedInvoiceURL, _ := event.Raw["hosted_invoice_url"].(string)
	invoicePDFURL, _ := event.Raw["invoice_pdf"].(string)

	subscriptionRef := event.SubscriptionRef
	if subscriptionRef == "" {
		subscriptionRef, _ = event.Raw["subscription"].(string)
	}

	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		inv, err := u.billingRepo.WithTx(tx).GetInvoiceByNumber(ctx, invoiceNumber)
		if err != nil && !errors.Is(err, domain.ErrInvoiceNotFound) {
			return err
		}

		now := time.Now()

		if errors.Is(err, domain.ErrInvoiceNotFound) {
			// Invoice not found locally — create it from Stripe event data.
			sub, lookupErr := u.subRepo.WithTx(tx).GetSubscriptionByStripeID(ctx, subscriptionRef)
			if lookupErr != nil {
				return fmt.Errorf("lookup subscription by stripe id %s: %w", subscriptionRef, lookupErr)
			}

			inv = &domain.Invoice{
				ID:              uuid.New(),
				EnterpriseID:    sub.EnterpriseID,
				SubscriptionID:  sub.ID,
				Number:          invoiceNumber,
				Status:          domain.InvoiceStatusPaid,
				AmountDue:       amountDue,
				AmountPaid:      amountPaid,
				AmountRemaining: 0,
				Currency:        currency,
				DueDate:         now,
				PaidAt:          &now,
				HostedInvoiceURL: &hostedInvoiceURL,
				InvoicePDFURL:   &invoicePDFURL,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := u.billingRepo.WithTx(tx).CreateInvoice(ctx, inv); err != nil {
				return err
			}
		} else {
			inv.Status = domain.InvoiceStatusPaid
			inv.AmountPaid = amountPaid
			inv.AmountRemaining = 0
			inv.PaidAt = &now
			if hostedInvoiceURL != "" {
				inv.HostedInvoiceURL = &hostedInvoiceURL
			}
			if invoicePDFURL != "" {
				inv.InvoicePDFURL = &invoicePDFURL
			}
			if err := u.billingRepo.WithTx(tx).UpdateInvoice(ctx, inv); err != nil {
				return err
			}
		}

		payment := &domain.Payment{
			ID:                uuid.New(),
			EnterpriseID:      inv.EnterpriseID,
			InvoiceID:         &inv.ID,
			Amount:            amountPaid,
			Currency:          currency,
			Status:            domain.PaymentStatusSucceeded,
			Provider:          domain.PaymentProviderStripe,
			ProviderPaymentID: event.EventID,
			CreatedAt:         time.Now(),
		}
		return u.billingRepo.WithTx(tx).CreatePayment(ctx, payment)
	})
}

func (u *paymentUsecase) handleInvoicePaymentFailed(ctx context.Context, event *domain.PaymentEvent) error {
	invoiceNumber, _ := event.Raw["number"].(string)

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

	// Publish Kafka event
	if enterpriseID != uuid.Nil && u.eventPublisher != nil {
		if err := u.eventPublisher.PublishPaymentFailed(ctx, enterpriseID); err != nil {
			zap.L().Error("handleInvoicePaymentFailed: failed to publish kafka event",
				zap.String("enterprise_id", enterpriseID.String()),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (u *paymentUsecase) handleInvoiceUpcoming(ctx context.Context, event *domain.PaymentEvent) error {
	subscriptionStr := event.SubscriptionRef
	if subscriptionStr == "" {
		subscriptionStr, _ = event.Raw["subscription"].(string)
	}

	if subscriptionStr == "" {
		return nil
	}

	sub, err := u.subRepo.GetSubscriptionByStripeID(ctx, subscriptionStr)
	if err != nil {
		if err == domain.ErrSubscriptionNotFound {
			return nil
		}
		return err
	}

	nextPaymentAttemptRaw, ok := event.Raw["next_payment_attempt"].(float64)
	if !ok {
		return nil
	}

	nextPaymentAttempt := time.Unix(int64(nextPaymentAttemptRaw), 0)
	daysUntil := int(time.Until(nextPaymentAttempt).Hours() / 24)

	// Publish notification event
	if u.eventPublisher != nil {
		return u.eventPublisher.PublishInvoiceUpcoming(ctx, sub.EnterpriseID, daysUntil)
	}
	return nil
}

func (u *paymentUsecase) handleSubscriptionUpdated(ctx context.Context, event *domain.PaymentEvent) error {
	stripeSubscriptionID, _ := event.Raw["id"].(string)
	if stripeSubscriptionID == "" {
		return nil
	}

	status, _ := event.Raw["status"].(string)
	currentPeriodStartRaw, _ := event.Raw["current_period_start"].(float64)
	currentPeriodEndRaw, _ := event.Raw["current_period_end"].(float64)
	cancelAtPeriodEnd, _ := event.Raw["cancel_at_period_end"].(bool)

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

		if u.eventPublisher != nil {
			if err := u.eventPublisher.PublishSubscriptionUpdated(ctx, sub.EnterpriseID); err != nil {
				zap.L().Error("failed to publish subscription_updated event", zap.Error(err))
			}
		}

		return nil
	})
}

func (u *paymentUsecase) handleSubscriptionDeleted(ctx context.Context, event *domain.PaymentEvent) error {
	stripeSubscriptionID, _ := event.Raw["id"].(string)
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

		if u.eventPublisher != nil {
			if err := u.eventPublisher.PublishSubscriptionCanceled(ctx, sub.EnterpriseID); err != nil {
				zap.L().Error("failed to publish subscription_canceled event", zap.Error(err))
			}
		}

		return nil
	})
}

// ─── Invoices & Payments ──────────────────────────────────────────────────────

func (u *paymentUsecase) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*domain.Invoice, error) {
	return u.billingRepo.GetInvoiceByID(ctx, invoiceID)
}

func (u *paymentUsecase) ListInvoices(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*domain.Invoice], error) {
	invoices, total, err := u.billingRepo.ListInvoicesByEnterprise(ctx, enterpriseID, params)
	if err != nil {
		return pagination.PaginatedResponse[*domain.Invoice]{}, err
	}
	return pagination.NewPaginatedResponse(invoices, total, params), nil
}

func (u *paymentUsecase) ListPaymentHistory(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*domain.Payment], error) {
	payments, total, err := u.billingRepo.ListPaymentsByEnterprise(ctx, enterpriseID, params)
	if err != nil {
		return pagination.PaginatedResponse[*domain.Payment]{}, err
	}
	return pagination.NewPaginatedResponse(payments, total, params), nil
}

func (u *paymentUsecase) RefundPayment(ctx context.Context, invoiceID uuid.UUID, amount float64, reason string) error {
	return RunInTx(ctx, u.pool, func(tx pgx.Tx) error {
		inv, err := u.billingRepo.WithTx(tx).GetInvoiceByID(ctx, invoiceID)
		if err != nil {
			return err
		}

		if inv.AmountPaid < amount {
			return fmt.Errorf("refund amount exceeds paid amount")
		}

		payment, err := u.billingRepo.WithTx(tx).GetPaymentByInvoiceID(ctx, invoiceID)
		if err != nil {
			return fmt.Errorf("fetch payment: %w", err)
		}

		prov, err := u.providerRegistry.Get(payment.Provider)
		if err != nil {
			return err
		}

		if err := prov.RefundPayment(ctx, payment.ProviderPaymentID, amount); err != nil {
			return err
		}

		refundPayment := &domain.Payment{
			ID:                uuid.New(),
			EnterpriseID:      inv.EnterpriseID,
			InvoiceID:         &inv.ID,
			Amount:            -amount,
			Currency:          inv.Currency,
			Status:            domain.PaymentStatusRefunded,
			Provider:          payment.Provider,
			ProviderPaymentID: payment.ProviderPaymentID,
			Notes:             &reason,
			CreatedAt:         time.Now(),
		}
		if err := u.billingRepo.WithTx(tx).CreatePayment(ctx, refundPayment); err != nil {
			return err
		}

		inv.AmountPaid -= amount
		inv.AmountRemaining += amount
		inv.UpdatedAt = time.Now()
		return u.billingRepo.WithTx(tx).UpdateInvoice(ctx, inv)
	})
}

// GetPayment retrieves a single payment record by its ID.
func (u *paymentUsecase) GetPayment(ctx context.Context, paymentID uuid.UUID) (*domain.Payment, error) {
	return u.billingRepo.GetPaymentByID(ctx, paymentID)
}

func (u *paymentUsecase) GetBillingSummary(ctx context.Context, enterpriseID uuid.UUID) (*domain.BillingSummary, error) {
	sub, err := u.GetActiveSubscription(ctx, enterpriseID)
	if err != nil && err != domain.ErrSubscriptionNotFound {
		return nil, fmt.Errorf("get active subscription: %w", err)
	}

	summary := &domain.BillingSummary{
		ActivePlanName:     "None",
		SubscriptionStatus: domain.SubStatusExpired,
		TotalPaidYTD:       0,
		OutstandingBalance: 0,
		LastPayment:        nil,
	}

	if sub != nil {
		summary.SubscriptionStatus = sub.Status
		if !sub.CurrentPeriodEnd.IsZero() {
			summary.NextBillingDate = &sub.CurrentPeriodEnd
		}

		plan, err := u.subRepo.GetPlanByID(ctx, sub.PlanID)
		if err == nil && plan != nil {
			summary.ActivePlanName = plan.Name
		}
	}

	totalPaid, outstanding, lastPayment, err := u.billingRepo.GetBillingAggregates(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("get billing aggregates: %w", err)
	}

	summary.TotalPaidYTD = totalPaid
	summary.OutstandingBalance = outstanding
	summary.LastPayment = lastPayment

	return summary, nil
}

func (u *paymentUsecase) calculatePeriodEnd(start time.Time, cycle domain.BillingCycle) time.Time {
	if cycle == domain.BillingCycleYearly {
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 1, 0)
}
