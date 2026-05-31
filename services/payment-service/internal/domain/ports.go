package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type SubscriptionRepository interface {
	ListPlans(ctx context.Context, params pagination.Params) ([]*SubscriptionPlan, int64, error)
	ListAllPlans(ctx context.Context, params pagination.Params) ([]*SubscriptionPlan, int64, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)

	GetSubscriptionByEnterpriseID(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	GetSubscriptionByStripeID(ctx context.Context, stripeSubscriptionID string) (*EnterpriseSubscription, error)
	GetSubscriptionByChapaTxRef(ctx context.Context, txRef string) (*EnterpriseSubscription, error)
	CreateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	UpdateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *SubscriptionPlan) error
	GetLapsedSubscriptions(ctx context.Context, limit int) ([]*EnterpriseSubscription, error)
	GetPastDueCandidates(ctx context.Context, limit int) ([]*EnterpriseSubscription, error)
	WithTx(tx pgx.Tx) SubscriptionRepository
}

type BillingRepository interface {
	CreateInvoice(ctx context.Context, inv *Invoice) error
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	GetInvoiceByNumber(ctx context.Context, number string) (*Invoice, error)
	ListInvoicesByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*Invoice, int64, error)
	UpdateInvoice(ctx context.Context, inv *Invoice) error

	CreatePayment(ctx context.Context, p *Payment) error
	GetPaymentByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*Payment, error)
	GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
	ListPaymentsByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*Payment, int64, error)

	RecordEventProcessed(ctx context.Context, eventID string, eventType string) error
	HasEventBeenProcessed(ctx context.Context, eventID string) (bool, error)

	GetBillingAggregates(ctx context.Context, enterpriseID uuid.UUID) (float64, float64, *Payment, error)
	GetOverdueInvoices(ctx context.Context, graceDays int, limit int) ([]*Invoice, error)
	PurgeOldWebhookEvents(ctx context.Context, cutoff time.Time) (int64, error)
	VoidOpenInvoices(ctx context.Context, enterpriseID uuid.UUID) (int64, error)
	WithTx(tx pgx.Tx) BillingRepository
}

// PaymentProvider is the provider-agnostic interface for payment gateway operations.
// Both Stripe and Chapa implement this interface.
type PaymentProvider interface {
	// CreateCheckoutSession returns a redirect URL for the user to complete payment.
	CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (string, error)

	// VerifyWebhookEvent parses and authenticates a raw webhook payload.
	// Returns a provider-agnostic PaymentEvent on success.
	VerifyWebhookEvent(payload []byte, sigHeader string) (*PaymentEvent, error)

	// CancelSubscription cancels an active provider-side subscription.
	// No-op for Chapa (no native recurring billing).
	CancelSubscription(ctx context.Context, providerSubID string, cancelAtPeriodEnd bool) error

	// ReactivateSubscription re-enables a pending period-end cancellation.
	// No-op for Chapa.
	ReactivateSubscription(ctx context.Context, providerSubID string) error

	// SyncPlan syncs plan pricing to the provider. Returns a provider-side price ID.
	// No-op for Chapa (returns "", nil).
	SyncPlan(ctx context.Context, plan *SubscriptionPlan) (string, error)

	// DeactivatePlan deactivates the plan price on the provider side.
	// No-op for Chapa.
	DeactivatePlan(ctx context.Context, providerPriceID string) error

	// RefundPayment issues a refund for a given provider payment reference.
	// Returns ErrNotSupported for Chapa in v1.
	RefundPayment(ctx context.Context, providerPaymentID string, amount float64) error
}

type PaymentUsecase interface {
	ListPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*SubscriptionPlan], error)
	ListAllPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*SubscriptionPlan], error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *SubscriptionPlan) error
	DeactivatePlan(ctx context.Context, planID uuid.UUID) error
	GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	// UpgradeSubscription creates a checkout URL. provider is "stripe" or "chapa" (default: "stripe").
	UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID, provider string) (string, error)

	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Invoice], error)
	ListPaymentHistory(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Payment], error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
	GetBillingSummary(ctx context.Context, enterpriseID uuid.UUID) (*BillingSummary, error)
	// HandleWebhook processes an incoming webhook event from the named provider.
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string, provider string) error
	CancelSubscription(ctx context.Context, enterpriseID uuid.UUID, cancelAtPeriodEnd bool) error
	ReactivateSubscription(ctx context.Context, enterpriseID uuid.UUID) error
	CreateTrialSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID, trialDays int) error
	AdminSetSubscription(ctx context.Context, enterpriseID uuid.UUID, req AdminSetSubscriptionRequest) error
	RefundPayment(ctx context.Context, invoiceID uuid.UUID, amount float64, reason string) error
}

type PaymentEventPublisher interface {
	PublishPaymentFailed(ctx context.Context, enterpriseID uuid.UUID) error
	PublishSubscriptionUpdated(ctx context.Context, enterpriseID uuid.UUID) error
	PublishSubscriptionCanceled(ctx context.Context, enterpriseID uuid.UUID) error
	PublishInvoiceUpcoming(ctx context.Context, enterpriseID uuid.UUID, daysUntil int) error
}