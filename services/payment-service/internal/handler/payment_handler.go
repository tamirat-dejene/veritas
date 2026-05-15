package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type PaymentHandler struct {
	usecase domain.PaymentUsecase
}

func NewPaymentHandler(u domain.PaymentUsecase) *PaymentHandler {
	return &PaymentHandler{usecase: u}
}

// ListPlans lists all active subscription plans.
//
//	@Summary		List subscription plans
//	@Description	Returns all available subscription plans.
//	@Tags			subscription
//	@Produce		json
//	@Param			page			query		int		false	"Page number (default 1)"
//	@Param			limit			query		int		false	"Items per page (default 10)"
//	@Param			sort			query		string	false	"Sort field (allowed: price, name, created_at; default: created_at)"
//	@Param			sort_dir		query		string	false	"Sort direction (asc/desc)"
//	@Success		200	{object}	pagination.PaginatedResponse[domain.SubscriptionPlan]
//	@Failure		500	{object}	ErrorResponse
//	@Router			/subscriptions/plans [get]
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	params := pagination.ParseGin(c)
	plans, err := h.usecase.ListPlans(c.Request.Context(), params)
	if err != nil {
		{ e := mapDomainError(err); writeError(c, e.Code, e.Message) }
		return
	}
	writeJSON(c, http.StatusOK, plans)
}

// UpgradeSubscription creates a checkout URL for plan upgrade.
//
//	@Summary		Upgrade enterprise subscription
//	@Description	Creates a provider checkout session for upgrading an enterprise subscription plan.
//	@Tags			subscription
//	@Accept			json
//	@Produce		json
//	@Param			enterpriseId	path		string					true	"Enterprise ID (UUID)"
//	@Param			body			body		UpgradeSubscriptionRequest	true	"Upgrade request"
//	@Success		200			{object}	CheckoutURLResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/subscriptions/{enterpriseId}/upgrade [post]
func (h *PaymentHandler) UpgradeSubscription(c *gin.Context) {
	enterpriseIDStr := c.Param("enterpriseId")
	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	var req UpgradeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid plan id")
		return
	}

	checkoutURL, err := h.usecase.UpgradeSubscription(c.Request.Context(), enterpriseID, planID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, CheckoutURLResponse{CheckoutURL: checkoutURL})
}

// HandleWebhook processes Stripe webhook events.
//
//	@Summary		Handle Stripe webhook
//	@Description	Validates and processes Stripe webhook event payload.
//	@Tags			webhook
//	@Accept			json
//	@Param			Stripe-Signature	header		string	true	"Stripe webhook signature"
//	@Param			payload				body		string	true	"Raw webhook payload"
//	@Success		200
//	@Failure		400	{object}	ErrorResponse
//	@Failure		503	{object}	ErrorResponse
//	@Router			/webhooks/stripe [post]
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		writeError(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	if err := h.usecase.HandleWebhook(c.Request.Context(), payload, sigHeader); err != nil {
		writeError(c, http.StatusServiceUnavailable, err.Error())
		return
	}

	c.Status(http.StatusOK)
}

// GetPayment retrieves a single payment by its ID.
//
//	@Summary		Get a single payment by ID
//	@Description	Enterprise admin or system admin retrieves a specific payment.
//	@Tags			payment
//	@Produce		json
//	@Param			paymentId path string true "Payment UUID"
//	@Success		200 {object} domain.Payment
//	@Failure		400 {object} ErrorResponse
//	@Failure		404 {object} ErrorResponse
//	@Failure		500 {object} ErrorResponse
//	@Router			/payments/{paymentId} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID, err := uuid.Parse(c.Param("paymentId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid payment id")
		return
	}

	payment, err := h.usecase.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, payment)
}

// ListPaymentHistory lists payment records for an enterprise.
//
//	@Summary		List payment history
//	@Description	Returns payment history for the specified enterprise.
//	@Tags			payment
//	@Produce		json
//	@Param			X-Enterprise-ID	header		string	true	"Enterprise ID"
//	@Param			page			query		int		false	"Page number (default 1)"
//	@Param			limit			query		int		false	"Items per page (default 10)"
//	@Param			sort			query		string	false	"Sort field (allowed: amount, status, payment_method_type, created_at; default: created_at)"
//	@Param			sort_dir		query		string	false	"Sort direction (asc/desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.Payment]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/payments/history [get]
func (h *PaymentHandler) ListPaymentHistory(c *gin.Context) {
	enterpriseIDStr := c.GetHeader("X-Enterprise-ID")
	if enterpriseIDStr == "" {
		writeError(c, http.StatusBadRequest, "X-Enterprise-ID header is required")
		return
	}

	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	params := pagination.ParseGin(c)

	payments, err := h.usecase.ListPaymentHistory(c.Request.Context(), enterpriseID, params)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, payments)
}

// ListInvoices lists invoices for an enterprise.
//
//	@Summary		List invoices
//	@Description	Returns invoices for the specified enterprise.
//	@Tags			payment
//	@Produce		json
//	@Param			X-Enterprise-ID	header		string	true	"Enterprise ID (UUID)"
//	@Param			page			query		int		false	"Page number (default 1)"
//	@Param			limit			query		int		false	"Items per page (default 10)"
//	@Param			sort			query		string	false	"Sort field (allowed: amount_due, amount_paid, amount_remaining, due_date, status, created_at; default: created_at)"
//	@Param			sort_dir		query		string	false	"Sort direction (asc/desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.Invoice]
//	@Failure		400				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/invoices [get]
func (h *PaymentHandler) ListInvoices(c *gin.Context) {
	enterpriseIDStr := c.GetHeader("X-Enterprise-ID")
	if enterpriseIDStr == "" {
		writeError(c, http.StatusBadRequest, "X-Enterprise-ID header is required")
		return
	}

	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	params := pagination.ParseGin(c)

	invoices, err := h.usecase.ListInvoices(c.Request.Context(), enterpriseID, params)
	if err != nil {
		{ e := mapDomainError(err); writeError(c, e.Code, e.Message) }
		return
	}

	writeJSON(c, http.StatusOK, invoices)
}

// GetInvoice returns invoice details by ID.
//
//	@Summary		Get invoice
//	@Description	Fetch a single invoice by invoice ID.
//	@Tags			payment
//	@Produce		json
//	@Param			invoiceId	path		string	true	"Invoice ID (UUID)"
//	@Success		200			{object}	domain.Invoice
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/invoices/{invoiceId} [get]
func (h *PaymentHandler) GetInvoice(c *gin.Context) {
	invoiceIDStr := c.Param("invoiceId")
	invoiceID, err := uuid.Parse(invoiceIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	invoice, err := h.usecase.GetInvoice(c.Request.Context(), invoiceID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, invoice)
}

// GetBillingSummary returns an aggregated billing summary for an enterprise.
//
//	@Summary		Get billing summary
//	@Description	Returns aggregated billing details including current plan, subscription status, and balances.
//	@Tags			billing
//	@Produce		json
//	@Param			X-Enterprise-ID	header		string	true	"Enterprise ID (UUID)"
//	@Success		200				{object}	domain.BillingSummary
//	@Failure		400				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/billing/summary [get]
func (h *PaymentHandler) GetBillingSummary(c *gin.Context) {
	enterpriseIDStr := c.GetHeader("X-Enterprise-ID")
	if enterpriseIDStr == "" {
		writeError(c, http.StatusBadRequest, "X-Enterprise-ID header is required")
		return
	}

	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	summary, err := h.usecase.GetBillingSummary(c.Request.Context(), enterpriseID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, summary)
}

// GetActiveSubscription returns the current subscription for an enterprise.
//
//	@Summary		Get active subscription
//	@Description	Returns the current subscription state for an enterprise.
//	@Tags			subscription
//	@Produce		json
//	@Param			enterpriseId	path		string	true	"Enterprise ID (UUID)"
//	@Success		200				{object}	domain.EnterpriseSubscription
//	@Failure		400				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/subscriptions/{enterpriseId} [get]
func (h *PaymentHandler) GetActiveSubscription(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}
	sub, err := h.usecase.GetActiveSubscription(c.Request.Context(), enterpriseID)
	if err != nil {
		handleError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, sub)
}

// CancelSubscription cancels an enterprise's subscription.
//
//	@Summary		Cancel subscription
//	@Description	Cancels an enterprise subscription immediately or at period end.
//	@Tags			subscription
//	@Accept			json
//	@Param			enterpriseId	path		string						true	"Enterprise ID (UUID)"
//	@Param			body			body		CancelSubscriptionRequest	false	"Cancel options"
//	@Success		204
//	@Failure		400				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		409				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/subscriptions/{enterpriseId}/cancel [post]
func (h *PaymentHandler) CancelSubscription(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}
	var req CancelSubscriptionRequest
	// body is optional — default is immediate cancel
	_ = c.ShouldBindJSON(&req)

	if err := h.usecase.CancelSubscription(c.Request.Context(), enterpriseID, req.CancelAtPeriodEnd); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ReactivateSubscription un-schedules a pending subscription cancellation.
//
//	@Summary		Reactivate subscription
//	@Description	Cancels a pending period-end cancellation, keeping the subscription active.
//	@Tags			subscription
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Success		204
//	@Failure		400				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/subscriptions/{enterpriseId}/reactivate [post]
func (h *PaymentHandler) ReactivateSubscription(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}
	if err := h.usecase.ReactivateSubscription(c.Request.Context(), enterpriseID); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AdminSetSubscription lets a system admin manually set an enterprise's subscription.
//
//	@Summary		Admin set subscription
//	@Description	Manually override an enterprise's subscription plan and status (no Stripe call).
//	@Tags			subscription
//	@Accept			json
//	@Param			enterpriseId	path		string						true	"Enterprise ID (UUID)"
//	@Param			body			body		AdminSetSubscriptionRequest	true	"Subscription override"
//	@Success		204
//	@Failure		400				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/admin/subscriptions/{enterpriseId} [post]
func (h *PaymentHandler) AdminSetSubscription(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}
	var req AdminSetSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid plan id")
		return
	}
	domainReq := domain.AdminSetSubscriptionRequest{
		PlanID:      planID,
		Status:      domain.SubscriptionStatus(req.Status),
		PeriodStart: req.PeriodStart,
		PeriodEnd:   req.PeriodEnd,
	}
	if err := h.usecase.AdminSetSubscription(c.Request.Context(), enterpriseID, domainReq); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreatePlan creates a new subscription plan.
//
//	@Summary		Create plan
//	@Description	System admin creates a new subscription plan.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreatePlanRequest	true	"Plan details"
//	@Success		201		{object}	domain.SubscriptionPlan
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/admin/plans [post]
func (h *PaymentHandler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := &domain.SubscriptionPlan{
		Name:          req.Name,
		Slug:          req.Slug,
		Description:   req.Description,
		Price:         req.Price,
		Currency:      domain.Currency(req.Currency),
		BillingCycle:  domain.BillingCycle(req.BillingCycle),
		Features:      req.Features,
		StripePriceID: req.StripePriceID,
		IsActive:      req.IsActive,
	}

	if err := h.usecase.CreatePlan(c.Request.Context(), plan); err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, plan)
}

// UpdatePlan updates an existing subscription plan.
//
//	@Summary		Update plan
//	@Description	System admin updates an existing subscription plan.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			planId	path		string				true	"Plan ID (UUID)"
//	@Param			body	body		UpdatePlanRequest	true	"Plan updates"
//	@Success		200		{object}	domain.SubscriptionPlan
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/admin/plans/{planId} [patch]
func (h *PaymentHandler) UpdatePlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("planId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid plan id")
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Fetch existing plan first to merge updates
	targetPlan, err := h.usecase.GetPlanByID(c.Request.Context(), planID)
	if err != nil {
		handleError(c, err)
		return
	}

	if req.Name != nil {
		targetPlan.Name = *req.Name
	}
	if req.Slug != nil {
		targetPlan.Slug = *req.Slug
	}
	if req.Description != nil {
		targetPlan.Description = *req.Description
	}
	if req.Price != nil {
		targetPlan.Price = *req.Price
	}
	if req.Currency != nil {
		targetPlan.Currency = domain.Currency(*req.Currency)
	}
	if req.BillingCycle != nil {
		targetPlan.BillingCycle = domain.BillingCycle(*req.BillingCycle)
	}
	if req.Features != nil {
		targetPlan.Features = req.Features
	}
	if req.StripePriceID != nil {
		targetPlan.StripePriceID = *req.StripePriceID
	}
	if req.IsActive != nil {
		targetPlan.IsActive = *req.IsActive
	}

	if err := h.usecase.UpdatePlan(c.Request.Context(), targetPlan); err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, targetPlan)
}

// AdminListPlans lists all subscription plans (including inactive).
//
//	@Summary		Admin list plans
//	@Description	Returns all subscription plans, including inactive ones.
//	@Tags			admin
//	@Produce		json
//	@Param			page			query		int		false	"Page number (default 1)"
//	@Param			limit			query		int		false	"Items per page (default 10)"
//	@Param			sort			query		string	false	"Sort field (allowed: price, name, created_at; default: created_at)"
//	@Param			sort_dir		query		string	false	"Sort direction (asc/desc)"
//	@Success		200	{object}	pagination.PaginatedResponse[domain.SubscriptionPlan]
//	@Failure		500	{object}	ErrorResponse
//	@Router			/admin/plans [get]
func (h *PaymentHandler) AdminListPlans(c *gin.Context) {
	params := pagination.ParseGin(c)
	plans, err := h.usecase.ListAllPlans(c.Request.Context(), params)
	if err != nil {
		{ e := mapDomainError(err); writeError(c, e.Code, e.Message) }
		return
	}
	writeJSON(c, http.StatusOK, plans)
}

// DeactivatePlan soft-deletes a plan by setting is_active = false.
//
//	@Summary		Deactivate plan
//	@Description	Sets is_active = false for a plan.
//	@Tags			admin
//	@Param			planId	path		string	true	"Plan ID (UUID)"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/admin/plans/{planId} [delete]
func (h *PaymentHandler) DeactivatePlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("planId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid plan id")
		return
	}

	if err := h.usecase.DeactivatePlan(c.Request.Context(), planID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RefundInvoice issues a refund for an invoice via Stripe.
//
//	@Summary		Refund invoice
//	@Description	Refunds a specific invoice by its ID.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			invoiceId	path		string			true	"Invoice ID (UUID)"
//	@Param			body		body		RefundRequest	true	"Refund request"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/admin/invoices/{invoiceId}/refund [post]
func (h *PaymentHandler) RefundInvoice(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("invoiceId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.usecase.RefundPayment(c.Request.Context(), invoiceID, req.Amount, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTrialSubscription provisions a free trial for an enterprise.
//
//	@Summary		Create trial subscription
//	@Description	Starts a free trial subscription without a payment method.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			enterpriseId	path		string				true	"Enterprise ID (UUID)"
//	@Param			body			body		CreateTrialRequest	true	"Trial request"
//	@Success		204
//	@Failure		400	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/admin/subscriptions/{enterpriseId}/trial [post]
func (h *PaymentHandler) CreateTrialSubscription(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	var req CreateTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid plan id")
		return
	}

	if err := h.usecase.CreateTrialSubscription(c.Request.Context(), enterpriseID, planID, req.TrialDays); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetFeatureGate returns the features available for the enterprise's active plan.
//
//	@Summary		Get plan features (Internal)
//	@Description	Returns the features JSONB for the active subscription plan. Used internally by other services.
//	@Tags			internal
//	@Produce		json
//	@Param			enterpriseId	path		string	true	"Enterprise ID (UUID)"
//	@Success		200	{object}	UsageResponse
//	@Failure		400	{object}	ErrorResponse
//	@Failure		404	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/billing/usage/{enterpriseId} [get]
func (h *PaymentHandler) GetFeatureGate(c *gin.Context) {
	enterpriseID, err := uuid.Parse(c.Param("enterpriseId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	sub, err := h.usecase.GetActiveSubscription(c.Request.Context(), enterpriseID)
	if err != nil {
		handleError(c, err)
		return
	}

	plan, err := h.usecase.GetPlanByID(c.Request.Context(), sub.PlanID)
	if err != nil {
		handleError(c, err)
		return
	}

	resp := UsageResponse{
		PlanFeatures: plan.Features,
	}

	writeJSON(c, http.StatusOK, resp)
}
