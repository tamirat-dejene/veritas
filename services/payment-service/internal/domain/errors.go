package domain

import "errors"

var (
	ErrPlanNotFound               = errors.New("subscription plan not found")
	ErrSubscriptionNotFound       = errors.New("subscription not found")
	ErrSubscriptionAlreadyCanceled = errors.New("subscription is already canceled")
	ErrInvoiceNotFound            = errors.New("invoice not found")
	ErrPaymentFailed              = errors.New("payment processing failed")
	ErrInvalidInput               = errors.New("invalid input")
	ErrUnauthorized               = errors.New("unauthorized")
	ErrForbidden                  = errors.New("forbidden")
	ErrInternal                   = errors.New("internal server error")
)
