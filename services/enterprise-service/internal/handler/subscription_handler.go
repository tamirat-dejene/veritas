package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// SubscriptionHandler handles subscription-related HTTP requests.
type SubscriptionHandler struct {
	usecase domain.EnterpriseUsecase
}

func NewSubscriptionHandler(uc domain.EnterpriseUsecase) *SubscriptionHandler {
	return &SubscriptionHandler{usecase: uc}
}

func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	var req domain.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.UpdateSubscription(c.Request.Context(), id, req, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.CancelSubscription(c.Request.Context(), id, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) RenewSubscription(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.RenewSubscription(c.Request.Context(), id, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) GetSubscriptionInfo(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	enterprise, err := h.usecase.GetSubscriptionInfo(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	// Return only subscription-relevant fields
	writeJSON(c, http.StatusOK, gin.H{
		"enterprise_id":        enterprise.ID,
		"subscription_plan_id": enterprise.SubscriptionPlanID,
		"subscription_status":  enterprise.SubscriptionStatus,
		"current_period_start": enterprise.CurrentPeriodStart,
		"current_period_end":   enterprise.CurrentPeriodEnd,
	})
}

func (h *SubscriptionHandler) SuspendForPayment(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.SuspendForPayment(c.Request.Context(), id, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) handleErr(c *gin.Context, err error) {
	switch err {
	case domain.ErrEnterpriseNotFound:
		writeError(c, http.StatusNotFound, "enterprise not found")
	case domain.ErrSubscriptionRequired:
		writeError(c, http.StatusConflict, "no subscription found")
	case domain.ErrInvalidStatus:
		writeError(c, http.StatusConflict, "invalid status transition")
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
