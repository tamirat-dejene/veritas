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

// UpdateSubscription updates enterprise subscription details.
//
//	@Summary		Update subscription
//	@Description	Update enterprise subscription plan/status/period fields.
//	@Tags			subscription
//	@Accept			json
//	@Param			enterpriseId	path	string					true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string					false	"Actor user ID (UUID)"
//	@Param			body			body	domain.UpdateSubscriptionRequest	true	"Subscription update payload"
//	@Success		204			{string}	string					"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/subscription [post]
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

// CancelSubscription cancels enterprise subscription.
//
//	@Summary		Cancel subscription
//	@Description	Cancel the current enterprise subscription.
//	@Tags			subscription
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/subscription/cancel [post]
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

// RenewSubscription renews enterprise subscription.
//
//	@Summary		Renew subscription
//	@Description	Renew the current enterprise subscription period.
//	@Tags			subscription
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/subscription/renew [post]
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

// GetSubscriptionInfo returns subscription-focused view for enterprise.
//
//	@Summary		Get subscription info
//	@Description	Return enterprise subscription details only.
//	@Tags			subscription
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Success		200			{object}	SubscriptionInfoResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/subscription [get]
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

// SuspendForPayment suspends an enterprise due to payment issues.
//
//	@Summary		Suspend for payment
//	@Description	Suspend enterprise subscription due to payment failure.
//	@Tags			subscription
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/suspend-payment [post]
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
