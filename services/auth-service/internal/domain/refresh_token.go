package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a row in the refresh_tokens table.
// Raw tokens are NEVER stored. Only the SHA-256 hash of the raw token is persisted.
type RefreshToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
}

// NewRefreshToken provides a standard way to initialize a RefreshToken.
func NewRefreshToken(userID uuid.UUID, tokenHash string, ttl time.Duration) *RefreshToken {
	return &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(ttl).UTC(),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
}
