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

func (h *PaymentHandler) ListPlans(c *gin.Context) {
	plans, err := h.usecase.ListPlans(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, plans)
}

func (h *PaymentHandler) UpgradeSubscription(c *gin.Context) {
	enterpriseIDStr := c.Param("enterpriseId")
	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise id")
		return
	}

	var req struct {
		PlanID uuid.UUID `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	checkoutURL, err := h.usecase.UpgradeSubscription(c.Request.Context(), enterpriseID, req.PlanID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"checkout_url": checkoutURL})
}

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
