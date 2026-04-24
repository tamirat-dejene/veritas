package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
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
//	@Success		200	{array}		domain.SubscriptionPlan
//	@Failure		500	{object}	ErrorResponse
//	@Router			/subscriptions/plans [get]
func (h *PaymentHandler) ListPlans(c *gin.Context) {
	plans, err := h.usecase.ListPlans(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
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
		writeError(c, http.StatusInternalServerError, err.Error())
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

// ListPaymentHistory lists payment records for an enterprise.
//
//	@Summary		List payment history
//	@Description	Returns payment history for the specified enterprise.
//	@Tags			payment
//	@Produce		json
//	@Param			X-Enterprise-ID	header		string	true	"Enterprise ID (UUID)"
//	@Success		200				{array}		domain.Payment
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

	payments, err := h.usecase.ListPaymentHistory(c.Request.Context(), enterpriseID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, payments)
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
		if err == domain.ErrInvoiceNotFound {
			writeError(c, http.StatusNotFound, err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, invoice)
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
		if err == domain.ErrSubscriptionNotFound {
			writeError(c, http.StatusNotFound, "no subscription found")
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
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
		switch err {
		case domain.ErrSubscriptionNotFound:
			writeError(c, http.StatusNotFound, "no subscription found")
		case domain.ErrSubscriptionAlreadyCanceled:
			writeError(c, http.StatusConflict, "subscription is already canceled")
		default:
			writeError(c, http.StatusInternalServerError, err.Error())
		}
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
		switch err {
		case domain.ErrSubscriptionNotFound:
			writeError(c, http.StatusNotFound, "no subscription found")
		default:
			writeError(c, http.StatusInternalServerError, err.Error())
		}
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
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
