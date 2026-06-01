package domain

import "errors"

var (
	ErrPlanNotFound               = errors.New("subscription plan not found")
	ErrPlanAlreadyExists          = errors.New("subscription plan with this name or slug already exists")
	ErrSubscriptionNotFound       = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists  = errors.New("subscription for this enterprise already exists")
	ErrSubscriptionAlreadyCanceled = errors.New("subscription is already canceled")
	ErrInvoiceNotFound            = errors.New("invoice not found")
	ErrInvoiceAlreadyExists       = errors.New("invoice with this number already exists")
	ErrPaymentNotFound            = errors.New("payment not found")
	ErrPaymentAlreadyExists       = errors.New("payment with this provider ID already exists")
	ErrPaymentFailed              = errors.New("payment processing failed")
	ErrInvalidInput               = errors.New("invalid input")
	ErrUnauthorized               = errors.New("unauthorized")
	ErrForbidden                  = errors.New("forbidden")
	ErrInternal                   = errors.New("internal server error")
	// ErrNotSupported is returned by provider operations that are not available
	// for a specific provider (e.g., Chapa refunds in v1).
	ErrNotSupported    = errors.New("operation not supported by this payment provider")
	// ErrInvalidProvider is returned when an unknown provider name is given.
	ErrInvalidProvider = errors.New("unknown payment provider")
)
