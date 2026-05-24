package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
)

// InternalSessionHandler exposes service-to-service endpoints for session data.
// These routes are intended for internal network use only (no public auth required).
type InternalSessionHandler struct {
	uc domain.InternalUseCase
}

func NewInternalSessionHandler(uc domain.InternalUseCase) *InternalSessionHandler {
	return &InternalSessionHandler{uc: uc}
}

// GetGradingPayload returns the full grading payload for a session, including
// session metadata, all session questions with evaluation criteria, and candidate answers.
// This endpoint is called by the grading-service after receiving the slim trigger event.
//
//	@Summary		Get grading payload
//	@Description	Internal endpoint — returns the aggregated grading data for a session.
//	@Tags			internal
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string	true	"Enterprise ID"
//	@Param			sessionId		path		string	true	"Session ID (UUID)"
//	@Success		200				{object}	domain.GradingPayload
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/internal/sessions/{sessionId}/grading-payload [get]
func (h *InternalSessionHandler) GetGradingPayload(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	enterpriseID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	payload, err := h.uc.BuildGradingPayload(c.Request.Context(), sessionID, enterpriseID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, payload)
}
