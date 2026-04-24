package domain

import (
	"context"

	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubscriptionRepository interface {
	ListPlans(ctx context.Context, params pagination.Params) ([]*SubscriptionPlan, int64, error)
	ListAllPlans(ctx context.Context, params pagination.Params) ([]*SubscriptionPlan, int64, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)

	GetSubscriptionByEnterpriseID(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	GetSubscriptionByStripeID(ctx context.Context, stripeSubscriptionID string) (*EnterpriseSubscription, error)
	CreateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	UpdateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *SubscriptionPlan) error
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

	WithTx(tx pgx.Tx) BillingRepository
}

type PaymentProvider interface {
	CreateCheckoutSession(ctx context.Context, enterpriseID uuid.UUID, plan *SubscriptionPlan, stripeCustomerID *string) (string, error)
	ConstructEvent(payload []byte, sigHeader string) (any, error)
	CancelStripeSubscription(ctx context.Context, stripeSubscriptionID string, cancelAtPeriodEnd bool) error
	ReactivateStripeSubscription(ctx context.Context, stripeSubscriptionID string) error
	RefundStripePayment(ctx context.Context, stripePaymentID string, amount float64) error
}

type PaymentUsecase interface {
	ListPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*SubscriptionPlan], error)
	ListAllPlans(ctx context.Context, params pagination.Params) (pagination.PaginatedResponse[*SubscriptionPlan], error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	CreatePlan(ctx context.Context, plan *SubscriptionPlan) error
	UpdatePlan(ctx context.Context, plan *SubscriptionPlan) error
	DeactivatePlan(ctx context.Context, planID uuid.UUID) error
	GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID) (string, error)

	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Invoice], error)
	ListPaymentHistory(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Payment], error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*Payment, error)
	GetBillingSummary(ctx context.Context, enterpriseID uuid.UUID) (*BillingSummary, error)
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error
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