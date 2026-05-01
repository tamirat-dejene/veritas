package handler

import "time"

// ErrorResponse is the standard error payload for this service.
type ErrorResponse struct {
	Error string `json:"error"`
}

// UpgradeSubscriptionRequest is the request body for subscription upgrades.
type UpgradeSubscriptionRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// CheckoutURLResponse is returned after creating a checkout session.
type CheckoutURLResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// CancelSubscriptionRequest is the request body for subscription cancellations.
type CancelSubscriptionRequest struct {
	// When true, the subscription remains active until the current period ends.
	// When false (default), it is canceled immediately.
	CancelAtPeriodEnd bool `json:"cancel_at_period_end"`
}

// AdminSetSubscriptionRequest is used by system admins to manually override
// an enterprise's subscription state (no Stripe call).
type AdminSetSubscriptionRequest struct {
	PlanID      string     `json:"plan_id"  binding:"required"`
	Status      string     `json:"status"   binding:"required"`
	PeriodStart *time.Time `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
}

type CreatePlanRequest struct {
	Name          string         `json:"name" binding:"required"`
	Slug          string         `json:"slug" binding:"required"`
	Description   string         `json:"description"`
	Price         float64        `json:"price" binding:"required"`
	Currency      string         `json:"currency" binding:"required"`
	BillingCycle  string         `json:"billing_cycle" binding:"required"`
	Features      map[string]any `json:"features"`
	StripePriceID string         `json:"stripe_price_id"`
	IsActive      bool           `json:"is_active"`
}

type UpdatePlanRequest struct {
	Name          *string         `json:"name"`
	Slug          *string         `json:"slug"`
	Description   *string         `json:"description"`
	Price         *float64        `json:"price"`
	Currency      *string         `json:"currency"`
	BillingCycle  *string         `json:"billing_cycle"`
	Features      map[string]any  `json:"features"`
	StripePriceID *string         `json:"stripe_price_id"`
	IsActive      *bool           `json:"is_active"`
}

// RefundRequest is the request body for refunding an invoice.
type RefundRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason"`
}

// CreateTrialRequest is the request body for starting a free trial without Stripe.
type CreateTrialRequest struct {
	PlanID    string `json:"plan_id" binding:"required"`
	TrialDays int    `json:"trial_days" binding:"required,gt=0"`
}

// UsageResponse is the response for internal feature gate checks.
type UsageResponse struct {
	PlanFeatures map[string]any `json:"plan_features"`
}
