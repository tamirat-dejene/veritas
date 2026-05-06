package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EnterpriseServiceClient is the interface for inter-service communication with the enterprise service.
type EnterpriseServiceClient interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
	UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error
}

// RefreshTokenRepository is the port for refresh token persistence operations.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	FindByHashForUpdate(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, tokenID uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	FindUsersWithExcessiveSessions(ctx context.Context, threshold int) ([]uuid.UUID, error)
	WithTx(tx pgx.Tx) RefreshTokenRepository
}

// TokenService is the port for token generation operations.
type TokenService interface {
	GenerateAccessToken(user *User) (tokenString string, err error)
	GenerateRefreshToken() (rawToken string, tokenHash string, err error)
}

// EventPublisher is the port for publishing domain events.
type EventPublisher interface {
	PublishLogin(ctx context.Context, userID uuid.UUID, email string) error
}
