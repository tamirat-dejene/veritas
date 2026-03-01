package domain

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrEnterpriseNotFound   = errors.New("enterprise not found")
	ErrAuditLogNotFound     = errors.New("audit log not found")
	ErrSlugAlreadyExists    = errors.New("enterprise slug already exists")
	ErrEmailAlreadyExists   = errors.New("user email already exists")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidInput         = errors.New("invalid input")
	ErrInternal             = errors.New("internal server error")
	ErrInvalidStatus        = errors.New("invalid status transition")
	ErrAlreadyActive        = errors.New("enterprise is already active")
	ErrSubscriptionRequired = errors.New("no active subscription found")
	ErrDomainValidation     = errors.New("domain validation failed")
	ErrRetentionActive      = errors.New("retention period has not yet expired")
	ErrInvalidRole          = errors.New("role not allowed for enterprise users")
	ErrCustomDomainInUse    = errors.New("custom domain already in use")
)
