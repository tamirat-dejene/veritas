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
//	@Param			enterpriseId	query		string	true	"Enterprise ID (UUID)"
//	@Success		200				{array}		domain.Payment
//	@Failure		400				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/payments/history [get]
func (h *PaymentHandler) ListPaymentHistory(c *gin.Context) {
	// In a real scenario, we'd extract enterpriseID from the authenticated user claims
	enterpriseIDStr := c.Query("enterpriseId")
	if enterpriseIDStr == "" {
		writeError(c, http.StatusBadRequest, "enterpriseId is required")
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
