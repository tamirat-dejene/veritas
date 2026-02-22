package domain

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEnterpriseNotFound    = errors.New("enterprise not found")
	ErrSlugAlreadyExists     = errors.New("enterprise slug already exists")
	ErrEmailAlreadyExists    = errors.New("user email already exists")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrInvalidInput          = errors.New("invalid input")
    ErrInternal              = errors.New("internal server error")
)
