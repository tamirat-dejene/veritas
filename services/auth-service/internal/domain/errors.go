package domain

import "errors"

// Sentinel errors used across all layers. Use errors.Is() to check them.

var (
	// ErrUserNotFound is returned when no user matches the given credentials.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidCredentials is returned when the supplied password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserInactive is returned when the user account is disabled.
	ErrUserInactive = errors.New("user account is inactive")

	// ErrUserDeleted is returned when the user account has been soft-deleted.
	ErrUserDeleted = errors.New("user account has been deleted")

	// ErrRoleNotPermitted is returned when a user's role is not allowed to use this service.
	ErrRoleNotPermitted = errors.New("role not permitted to authenticate via this service")

	// ErrTokenNotFound is returned when a refresh token cannot be located by its hash.
	ErrTokenNotFound = errors.New("refresh token not found")

	// ErrTokenRevoked is returned when a refresh token has already been revoked.
	ErrTokenRevoked = errors.New("refresh token has been revoked")

	// ErrTokenExpired is returned when a refresh token is past its expiry time.
	ErrTokenExpired = errors.New("refresh token has expired")
)
