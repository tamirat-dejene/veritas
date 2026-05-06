package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type passwordResetRepository struct {
	db DBTX
}

// NewPasswordResetRepository creates a new PasswordResetRepository.
func NewPasswordResetRepository(db DBTX) domain.PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) WithTx(tx pgx.Tx) domain.PasswordResetRepository {
	return &passwordResetRepository{db: tx}
}

// CreateToken inserts a new (hashed) password reset token for the given user.
func (r *passwordResetRepository) CreateToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt)
	return err
}

// FindByTokenHash retrieves an un-used token record by its SHA-256 hash.
// Returns domain.ErrResetTokenInvalid if no matching, active record is found.
func (r *passwordResetRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		LIMIT 1
	`
	row := r.db.QueryRow(ctx, query, tokenHash)

	var t domain.PasswordResetToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Used, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrResetTokenInvalid
		}
		return nil, err
	}
	return &t, nil
}

// InvalidatePreviousTokens marks all outstanding tokens for a user as used,
// ensuring only one active reset token exists at any time.
func (r *passwordResetRepository) InvalidatePreviousTokens(ctx context.Context, userID uuid.UUID) error {
	const query = `
		UPDATE password_reset_tokens
		SET used = true
		WHERE user_id = $1 AND used = false
	`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// MarkUsed marks a single token as consumed so it cannot be reused.
func (r *passwordResetRepository) MarkUsed(ctx context.Context, tokenID uuid.UUID) error {
	const query = `UPDATE password_reset_tokens SET used = true WHERE id = $1`
	_, err := r.db.Exec(ctx, query, tokenID)
	return err
}

func (r *passwordResetRepository) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	const query = `DELETE FROM password_reset_tokens WHERE expires_at <= NOW() OR used = true`
	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
