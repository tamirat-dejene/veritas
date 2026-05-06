package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

// refreshTokenRepository implements domain.RefreshTokenRepository backed by PostgreSQL.
type refreshTokenRepository struct {
	db DBTX
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(db DBTX) domain.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) WithTx(tx pgx.Tx) domain.RefreshTokenRepository {
	return &refreshTokenRepository{db: tx}
}

// Create persists a new refresh token record.
func (r *refreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.Revoked,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("refreshTokenRepository.Create: %w", err)
	}
	return nil
}

// FindByHash looks up a refresh token by its SHA-256 hash.
// Returns domain.ErrTokenNotFound when no matching row exists.
func (r *refreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var t domain.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.ExpiresAt,
		&t.Revoked,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("refreshTokenRepository.FindByHash: %w", err)
	}

	return &t, nil
}

// FindByHashForUpdate looks up a refresh token by hash and acquires a row lock.
// Returns domain.ErrTokenNotFound when no matching row exists.
func (r *refreshTokenRepository) FindByHashForUpdate(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
		FOR UPDATE
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var t domain.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.ExpiresAt,
		&t.Revoked,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("refreshTokenRepository.FindByHashForUpdate: %w", err)
	}

	return &t, nil
}

// Revoke marks the token with the given ID as revoked.
func (r *refreshTokenRepository) Revoke(ctx context.Context, tokenID uuid.UUID) error {
	const query = `UPDATE refresh_tokens SET revoked = true WHERE id = $1 AND revoked = false`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	commandTag, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("refreshTokenRepository.Revoke: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrTokenRevoked
	}
	return nil
}

// DeleteExpired removes all tokens that have expired before the given time.
func (r *refreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	const query = `DELETE FROM refresh_tokens WHERE expires_at < $1`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // Cleanup can take longer
	defer cancel()

	commandTag, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("refreshTokenRepository.DeleteExpired: %w", err)
	}
	return commandTag.RowsAffected(), nil
}

// FindUsersWithExcessiveSessions identifies users with more than 'threshold' active refresh tokens.
func (r *refreshTokenRepository) FindUsersWithExcessiveSessions(ctx context.Context, threshold int) ([]uuid.UUID, error) {
	const query = `
		SELECT user_id
		FROM refresh_tokens
		WHERE revoked = false AND expires_at > NOW()
		GROUP BY user_id
		HAVING COUNT(*) > $1
	`

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.db.Query(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("refreshTokenRepository.FindUsersWithExcessiveSessions: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user_id: %w", err)
		}
		userIDs = append(userIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return userIDs, nil
}
