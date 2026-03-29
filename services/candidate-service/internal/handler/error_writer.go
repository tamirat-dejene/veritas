package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

// HandleError centralizes domain error mapping to HTTP status codes.
func HandleError(c *gin.Context, l *zap.Logger, err error) {
	if err == nil {
		return
	}

	statusCode := http.StatusInternalServerError
	message := "Internal server error"

	switch {
	case errors.Is(err, domain.ErrCandidateNotFound),
		errors.Is(err, domain.ErrEnrollmentNotFound),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrQuestionNotFound),
		errors.Is(err, domain.ErrSubmissionNotFound):
		statusCode = http.StatusNotFound
		message = err.Error()

	case errors.Is(err, domain.ErrUnauthorizedCandidate),
		errors.Is(err, domain.ErrInvalidAccessToken):
		statusCode = http.StatusUnauthorized
		message = err.Error()

	case errors.Is(err, domain.ErrUnauthorizedAccess):
		statusCode = http.StatusForbidden
		message = err.Error()

	case errors.Is(err, domain.ErrMaxAttemptsReached),
		errors.Is(err, domain.ErrSessionNotActive),
		errors.Is(err, domain.ErrSessionTerminated),
		errors.Is(err, domain.ErrSessionAlreadySubmitted),
		errors.Is(err, domain.ErrSessionAlreadyActive),
		errors.Is(err, domain.ErrSessionExpired),
		errors.Is(err, domain.ErrExamNotScheduled):
		statusCode = http.StatusForbidden
		message = err.Error()

	case errors.Is(err, domain.ErrInvalidAnswerFormat):
		statusCode = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, domain.ErrDuplicateExternalID),
		errors.Is(err, domain.ErrSubmissionExists):
		statusCode = http.StatusConflict
		message = err.Error()
	}

	if statusCode == http.StatusInternalServerError {
		logger.WithContext(c.Request.Context(), l).Error("Unexpected error", zap.Error(err))
	} else {
		logger.WithContext(c.Request.Context(), l).Warn("Request failed", zap.Error(err), zap.Int("status", statusCode))
	}

	c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
}
