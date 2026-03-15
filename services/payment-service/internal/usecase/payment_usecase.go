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
)

type paymentUsecase struct {
	pool        *pgxpool.Pool
	subRepo     domain.SubscriptionRepository
	billingRepo domain.BillingRepository
	payProvider domain.PaymentProvider
}

func NewPaymentUsecase(pool *pgxpool.Pool, subRepo domain.SubscriptionRepository, billingRepo domain.BillingRepository, payProvider domain.PaymentProvider) domain.PaymentUsecase {
	return &paymentUsecase{
		pool:        pool,
		subRepo:     subRepo,
		billingRepo: billingRepo,
		payProvider: payProvider,
	}
}

func (u *paymentUsecase) ListPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	return u.subRepo.ListPlans(ctx)
}

func (u *paymentUsecase) GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*domain.EnterpriseSubscription, error) {
	return u.subRepo.GetSubscriptionByEnterpriseID(ctx, enterpriseID)
}

func (u *paymentUsecase) UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID) (string, error) {
	plan, err := u.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return "", err
	}

	// In a Stripe-based flow, we create a checkout session
	checkoutURL, err := u.payProvider.CreateCheckoutSession(ctx, enterpriseID, plan)
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

func (u *paymentUsecase) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	eventAny, err := u.payProvider.ConstructEvent(payload, sigHeader)
	if err != nil {
		return err
	}

	event, ok := eventAny.(*stripe.Event)
	if !ok {
		return fmt.Errorf("unexpected event type")
	}

	switch event.Type {
	case "checkout.session.completed":
		// Handle initial subscription completion
		return u.handleCheckoutSessionCompleted(ctx, event)
	case "invoice.paid":
		// Handle recurring payment success
		return u.handleInvoicePaid(ctx, event)
	case "invoice.payment_failed":
		// Handle payment failure
		return u.handleInvoicePaymentFailed(ctx, event)
	}

	return nil
}

func (u *paymentUsecase) handleCheckoutSessionCompleted(ctx context.Context, event *stripe.Event) error {
	// Logic to extract metadata and update subscription status
	metadata := event.Data.Object["metadata"].(map[string]any)
	enterpriseIDStr := metadata["enterprise_id"].(string)
	planIDStr := metadata["plan_id"].(string)

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
				return u.subRepo.WithTx(tx).CreateSubscription(ctx, newSub)
			}
			return err
		}

		sub.PlanID = planID
		sub.Status = domain.SubStatusActive
		sub.UpdatedAt = time.Now()
		return u.subRepo.WithTx(tx).UpdateSubscription(ctx, sub)
	})
}

func (u *paymentUsecase) handleInvoicePaid(ctx context.Context, event *stripe.Event) error {
	// Extract invoice data and record payment
	invoiceObj := event.Data.Object
	invoiceNumber := invoiceObj["number"].(string)
	amountPaid := float64(invoiceObj["amount_paid"].(int64)) / 100
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

func (u *paymentUsecase) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	// Logic to mark invoice as failed and notify user
	return nil
}

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
