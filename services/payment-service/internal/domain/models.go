package domain

import (
	"time"

	"github.com/google/uuid"
)

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

type Currency string

const (
	CurrencyETB Currency = "ETB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
)

type SubscriptionStatus string

const (
	SubStatusActive   SubscriptionStatus = "Active"
	SubStatusPastDue  SubscriptionStatus = "PastDue"
	SubStatusCanceled SubscriptionStatus = "Canceled"
	SubStatusExpired  SubscriptionStatus = "Expired"
	SubStatusTrial    SubscriptionStatus = "Trial"
)

type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "Draft"
	InvoiceStatusOpen          InvoiceStatus = "Open"
	InvoiceStatusPaid          InvoiceStatus = "Paid"
	InvoiceStatusVoid          InvoiceStatus = "Void"
	InvoiceStatusUncollectible InvoiceStatus = "Uncollectible"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "Pending"
	PaymentStatusSucceeded PaymentStatus = "Succeeded"
	PaymentStatusFailed    PaymentStatus = "Failed"
	PaymentStatusRefunded  PaymentStatus = "Refunded"
)

type SubscriptionPlan struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	Name          string         `db:"name" json:"name"`
	Slug          string         `db:"slug" json:"slug"`
	Description   string         `db:"description" json:"description"`
	Price         float64        `db:"price" json:"price"`
	Currency      Currency       `db:"currency" json:"currency"`
	BillingCycle  BillingCycle   `db:"billing_cycle" json:"billing_cycle"`
	Features      map[string]any `db:"features" json:"features"`
	StripePriceID string         `db:"stripe_price_id" json:"stripe_price_id"`
	IsActive      bool           `db:"is_active" json:"is_active"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}

type EnterpriseSubscription struct {
	ID                   uuid.UUID          `db:"id" json:"id"`
	EnterpriseID         uuid.UUID          `db:"enterprise_id" json:"enterprise_id"`
	PlanID               uuid.UUID          `db:"plan_id" json:"plan_id"`
	Status               SubscriptionStatus `db:"status" json:"status"`
	CurrentPeriodStart   time.Time          `db:"current_period_start" json:"current_period_start"`
	CurrentPeriodEnd     time.Time          `db:"current_period_end" json:"current_period_end"`
	CancelAtPeriodEnd    bool               `db:"cancel_at_period_end" json:"cancel_at_period_end"`
	CanceledAt           *time.Time         `db:"canceled_at" json:"canceled_at,omitempty"`
	EndedAt              *time.Time         `db:"ended_at" json:"ended_at,omitempty"`
	StripeCustomerID     *string            `db:"stripe_customer_id" json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID *string            `db:"stripe_subscription_id" json:"stripe_subscription_id,omitempty"`
	CreatedAt            time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time          `db:"updated_at" json:"updated_at"`
}

type Invoice struct {
	ID               uuid.UUID     `db:"id" json:"id"`
	EnterpriseID     uuid.UUID     `db:"enterprise_id" json:"enterprise_id"`
	SubscriptionID   uuid.UUID     `db:"subscription_id" json:"subscription_id"`
	Number           string        `db:"number" json:"number"`
	Status           InvoiceStatus `db:"status" json:"status"`
	AmountDue        float64       `db:"amount_due" json:"amount_due"`
	AmountPaid       float64       `db:"amount_paid" json:"amount_paid"`
	AmountRemaining  float64       `db:"amount_remaining" json:"amount_remaining"`
	Currency         Currency      `db:"currency" json:"currency"`
	DueDate          time.Time     `db:"due_date" json:"due_date"`
	PaidAt           *time.Time    `db:"paid_at" json:"paid_at,omitempty"`
	HostedInvoiceURL *string       `db:"hosted_invoice_url" json:"hosted_invoice_url,omitempty"`
	InvoicePDFURL    *string       `db:"invoice_pdf_url" json:"invoice_pdf_url,omitempty"`
	CreatedAt        time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time     `db:"updated_at" json:"updated_at"`
}

type Payment struct {
	ID                   uuid.UUID     `db:"id" json:"id"`
	EnterpriseID         uuid.UUID     `db:"enterprise_id" json:"enterprise_id"`
	InvoiceID            *uuid.UUID    `db:"invoice_id" json:"invoice_id,omitempty"`
	Amount               float64       `db:"amount" json:"amount"`
	Currency             Currency      `db:"currency" json:"currency"`
	Status               PaymentStatus `db:"status" json:"status"`
	PaymentMethodType    *string       `db:"payment_method_type" json:"payment_method_type,omitempty"`
	Provider             string        `db:"provider" json:"provider"`
	ProviderPaymentID    string        `db:"provider_payment_id" json:"provider_payment_id"`
	ProviderErrorCode    *string       `db:"provider_error_code" json:"provider_error_code,omitempty"`
	ProviderErrorMessage *string       `db:"provider_error_message" json:"provider_error_message,omitempty"`
	Notes                *string       `db:"notes" json:"notes,omitempty"`
	CreatedAt            time.Time     `db:"created_at" json:"created_at"`
}

// AdminSetSubscriptionRequest is used by system admins to manually override
// an enterprise's subscription state without going through Stripe.
type AdminSetSubscriptionRequest struct {
	PlanID      uuid.UUID          `json:"plan_id"`
	Status      SubscriptionStatus `json:"status"`
	PeriodStart *time.Time         `json:"period_start,omitempty"`
	PeriodEnd   *time.Time         `json:"period_end,omitempty"`
}

type BillingSummary struct {
	ActivePlanName     string             `json:"active_plan_name"`
	SubscriptionStatus SubscriptionStatus `json:"subscription_status"`
	NextBillingDate    *time.Time         `json:"next_billing_date"`
	TotalPaidYTD       float64            `json:"total_paid_ytd"`
	OutstandingBalance float64            `json:"outstanding_balance"`
	LastPayment        *Payment           `json:"last_payment"`
}
