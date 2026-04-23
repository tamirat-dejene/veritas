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
