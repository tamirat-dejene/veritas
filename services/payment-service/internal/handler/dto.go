package handler

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
