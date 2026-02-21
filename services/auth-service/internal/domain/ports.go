package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository is the port for user persistence operations.
type UserRepository interface {
	// FindByEmail returns a user by email address.
	// Returns ErrUserNotFound if no user exists with that email.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByID returns a user by primary key.
	// Returns ErrUserNotFound if no user exists with that ID.
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
}

// RefreshTokenRepository is the port for refresh token persistence operations.
type RefreshTokenRepository interface {
	// Create persists a new refresh token.
	Create(ctx context.Context, token *RefreshToken) error

	// FindByHash looks up a refresh token by its SHA-256 hash.
	// Returns ErrTokenNotFound if no matching token exists.
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// Revoke marks the token with the given ID as revoked.
	Revoke(ctx context.Context, tokenID uuid.UUID) error

	// DeleteExpiredByUserID removes all expired tokens for a user (optional housekeeping).
	DeleteExpiredByUserID(ctx context.Context, userID uuid.UUID, before time.Time) error
}

// TokenService is the port for token generation operations.
type TokenService interface {
	// GenerateAccessToken produces a signed JWT for the given user.
	GenerateAccessToken(user *User) (tokenString string, err error)

	// GenerateRefreshToken produces a cryptographically secure random raw token
	// and its SHA-256 hash suitable for storage.
	GenerateRefreshToken() (rawToken string, tokenHash string, err error)
}
