package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
)

// HandleError centralizes domain error mapping to HTTP status codes.
func HandleError(c *gin.Context, err error) {
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
		errors.Is(err, domain.ErrInvalidAccessToken),
		errors.Is(err, domain.ErrEnterpriseIDMissing),
		errors.Is(err, domain.ErrEnrollmentIDMissing),
		errors.Is(err, domain.ErrCandidateIDMissing),
		errors.Is(err, domain.ErrUnauthorizedContext),
		errors.Is(err, domain.ErrInvalidToken):
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
		errors.Is(err, domain.ErrExamNotScheduled),
		errors.Is(err, domain.ErrInvalidExamStatus),
		errors.Is(err, domain.ErrInvalidEnrollmentTime):
		statusCode = http.StatusForbidden
		message = err.Error()

	case errors.Is(err, domain.ErrInvalidAnswerFormat),
		errors.Is(err, domain.ErrInvalidIDFormat),
		errors.Is(err, domain.ErrMissingFile),
		errors.Is(err, domain.ErrNoValidCandidates),
		errors.Is(err, domain.ErrNotAString):
		statusCode = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, domain.ErrDuplicateExternalID),
		errors.Is(err, domain.ErrSubmissionExists):
		statusCode = http.StatusConflict
		message = err.Error()

	case errors.Is(err, domain.ErrFileTooLarge):
		statusCode = http.StatusRequestEntityTooLarge
		message = err.Error()

	case errors.Is(err, domain.ErrNotSupported):
		statusCode = http.StatusNotImplemented
		message = err.Error()
	}

	c.AbortWithStatusJSON(statusCode, dto.ErrorResponse{Error: message})
}
