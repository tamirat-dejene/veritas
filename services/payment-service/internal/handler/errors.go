package handler

import (
	"errors"
	"net/http"

	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

// httpError pairs an HTTP status code with a safe client-facing message.
type httpError struct {
	Code    int
	Message string
}

// mapDomainError translates domain sentinel errors into appropriate HTTP status
// codes and safe, non-leaking messages. Unknown errors default to 500.
func mapDomainError(err error) httpError {
	switch {
	case errors.Is(err, domain.ErrPlanNotFound):
		return httpError{http.StatusNotFound, "subscription plan not found"}
	case errors.Is(err, domain.ErrPlanAlreadyExists):
		return httpError{http.StatusConflict, "subscription plan with this name or slug already exists"}
	case errors.Is(err, domain.ErrSubscriptionNotFound):
		return httpError{http.StatusNotFound, "subscription not found"}
	case errors.Is(err, domain.ErrSubscriptionAlreadyExists):
		return httpError{http.StatusConflict, "subscription for this enterprise already exists"}
	case errors.Is(err, domain.ErrSubscriptionAlreadyCanceled):
		return httpError{http.StatusConflict, "subscription is already canceled"}
	case errors.Is(err, domain.ErrInvoiceNotFound):
		return httpError{http.StatusNotFound, "invoice not found"}
	case errors.Is(err, domain.ErrInvoiceAlreadyExists):
		return httpError{http.StatusConflict, "invoice with this number already exists"}
	case errors.Is(err, domain.ErrPaymentNotFound):
		return httpError{http.StatusNotFound, "payment not found"}
	case errors.Is(err, domain.ErrPaymentAlreadyExists):
		return httpError{http.StatusConflict, "payment with this provider ID already exists"}
	case errors.Is(err, domain.ErrPaymentFailed):
		return httpError{http.StatusPaymentRequired, "payment processing failed"}
	case errors.Is(err, domain.ErrInvalidInput):
		return httpError{http.StatusBadRequest, "invalid input"}
	case errors.Is(err, domain.ErrUnauthorized):
		return httpError{http.StatusUnauthorized, "unauthorized"}
	case errors.Is(err, domain.ErrForbidden):
		return httpError{http.StatusForbidden, "forbidden"}
	default:
		return httpError{http.StatusInternalServerError, "an internal error occurred"}
	}
}
