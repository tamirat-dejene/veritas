package domain

import "errors"

// Sentinel errors used across all layers.

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrUserDeleted        = errors.New("user account has been deleted")
	ErrRoleNotPermitted   = errors.New("role not permitted to authenticate via this service")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrTokenRevoked       = errors.New("refresh token has been revoked")
	ErrTokenExpired       = errors.New("refresh token has expired")
	ErrAccountLocked = errors.New("user account is locked")
)
