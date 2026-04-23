package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SubscriptionRepository interface {
	ListPlans(ctx context.Context) ([]*SubscriptionPlan, error)
	GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error)
	GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error)

	GetSubscriptionByEnterpriseID(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	CreateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	UpdateSubscription(ctx context.Context, sub *EnterpriseSubscription) error
	WithTx(tx pgx.Tx) SubscriptionRepository
}

type BillingRepository interface {
	CreateInvoice(ctx context.Context, inv *Invoice) error
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	GetInvoiceByNumber(ctx context.Context, number string) (*Invoice, error)
	ListInvoicesByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*Invoice, error)
	UpdateInvoice(ctx context.Context, inv *Invoice) error

	CreatePayment(ctx context.Context, p *Payment) error
	ListPaymentsByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*Payment, error)
	WithTx(tx pgx.Tx) BillingRepository
}

type PaymentProvider interface {
	CreateCheckoutSession(ctx context.Context, enterpriseID uuid.UUID, plan *SubscriptionPlan) (string, error)
	ConstructEvent(payload []byte, sigHeader string) (any, error)
	CancelStripeSubscription(ctx context.Context, stripeSubscriptionID string, cancelAtPeriodEnd bool) error
	ReactivateStripeSubscription(ctx context.Context, stripeSubscriptionID string) error
}

type PaymentUsecase interface {
	ListPlans(ctx context.Context) ([]*SubscriptionPlan, error)
	GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*EnterpriseSubscription, error)
	UpgradeSubscription(ctx context.Context, enterpriseID uuid.UUID, planID uuid.UUID) (string, error)

	GetInvoice(ctx context.Context, invoiceID uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, enterpriseID uuid.UUID) ([]*Invoice, error)
	ListPaymentHistory(ctx context.Context, enterpriseID uuid.UUID) ([]*Payment, error)
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error
	CancelSubscription(ctx context.Context, enterpriseID uuid.UUID, cancelAtPeriodEnd bool) error
	ReactivateSubscription(ctx context.Context, enterpriseID uuid.UUID) error
	AdminSetSubscription(ctx context.Context, enterpriseID uuid.UUID, req AdminSetSubscriptionRequest) error
}

type PaymentEventPublisher interface {
	PublishPaymentFailed(ctx context.Context, enterpriseID uuid.UUID) error
}