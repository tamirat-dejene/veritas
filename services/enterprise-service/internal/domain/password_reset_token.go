package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a single-use, time-limited password reset record.
// The raw token is never persisted — only its SHA-256 hex digest (TokenHash) is stored.
type PasswordResetToken struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	Used      bool      `db:"used"`
	CreatedAt time.Time `db:"created_at"`
}
