package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"go.uber.org/zap"
)

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	log := zap.L().With(zap.String("path", c.FullPath()))

	if errors.Is(err, domain.ErrExamNotFound) || errors.Is(err, domain.ErrQuestionNotFound) || errors.Is(err, domain.ErrMappingNotFound) {
		log.Warn("resource not found", zap.Error(err))
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if errors.Is(err, domain.ErrInvalidStatus) || errors.Is(err, domain.ErrDuplicateOrderIndex) {
		log.Warn("conflict", zap.Error(err))
		writeError(c, http.StatusConflict, err.Error())
		return
	}

	if errors.Is(err, domain.ErrInvalidOrderIndex) || errors.Is(err, domain.ErrOrderIndexGap) ||
		errors.Is(err, domain.ErrInvalidQuestion) || errors.Is(err, domain.ErrNoQuestions) ||
		errors.Is(err, domain.ErrQuestionValidationFailed) || errors.Is(err, domain.ErrInsufficientTime) ||
		errors.Is(err, domain.ErrExamCannotBeDeleted) || errors.Is(err, domain.ErrInvalidTime) {
		log.Warn("bad request", zap.Error(err))
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if errors.Is(err, domain.ErrUnauthorized) {
		log.Warn("unauthorized", zap.Error(err))
		writeError(c, http.StatusForbidden, err.Error())
		return
	}

	if errors.Is(err, domain.ErrMarshalFailed) || errors.Is(err, domain.ErrInternal) {
		log.Error("internal error", zap.Error(err))
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	log.Error("unhandled error", zap.Error(err))
	writeError(c, http.StatusInternalServerError, "internal server error")
}

func writeJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

// Helper to extract EnterpriseID and UserID from context
// In a real scenario, these come from middleware set by the Gateway
func getEnterpriseID(c *gin.Context) (uuid.UUID, bool) {
	idStr := c.GetHeader("X-Enterprise-ID")
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}

func getUserID(c *gin.Context) (uuid.UUID, bool) {
	idStr := c.GetHeader("X-User-ID")
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}
