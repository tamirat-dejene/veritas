package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
)

type userRepository struct {
	db postgres.PostgresClient
}

func NewUserRepository(db postgres.PostgresClient) domain.UserRepository {
	return &userRepository{db: db}
}

const userFields = `
	id, email, password_hash, honorific, first_name, last_name, phone, role, enterprise_id,
	is_active, is_deleted, email_verified, email_verified_at, failed_login_attempts, locked_until,
	password_changed_at, must_change_password, last_login_at, last_login_ip, last_user_agent,
	created_at, updated_at
`

func scanUser(row postgres.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Honorific, &u.FirstName, &u.LastName, &u.Phone, &u.Role, &u.EnterpriseID,
		&u.IsActive, &u.IsDeleted, &u.EmailVerified, &u.EmailVerifiedAt, &u.FailedLoginAttempts, &u.LockedUntil,
		&u.PasswordChangedAt, &u.MustChangePassword, &u.LastLoginAt, &u.LastLoginIP, &u.LastUserAgent,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	const query = `
		INSERT INTO veritas_users (
			id, email, password_hash, honorific, first_name, last_name, phone, role, enterprise_id,
			is_active, is_deleted, email_verified, email_verified_at, failed_login_attempts, locked_until,
			password_changed_at, must_change_password, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	if u.PasswordChangedAt.IsZero() {
		u.PasswordChangedAt = now
	}

	_, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.PasswordHash, u.Honorific, u.FirstName, u.LastName, u.Phone, u.Role, u.EnterpriseID,
		u.IsActive, u.IsDeleted, u.EmailVerified, u.EmailVerifiedAt, u.FailedLoginAttempts, u.LockedUntil,
		u.PasswordChangedAt, u.MustChangePassword, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_users WHERE id = $1 LIMIT 1", userFields)
	row := r.db.QueryRow(ctx, query, id)
	return scanUser(row)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_users WHERE email = $1 LIMIT 1", userFields)
	row := r.db.QueryRow(ctx, query, email)
	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, u *domain.User) error {
	const query = `
		UPDATE veritas_users
		SET email = $2, password_hash = $3, honorific = $4, first_name = $5, last_name = $6, phone = $7,
		    role = $8, enterprise_id = $9, is_active = $10, is_deleted = $11, email_verified = $12,
		    email_verified_at = $13, failed_login_attempts = $14, locked_until = $15,
		    password_changed_at = $16, must_change_password = $17, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.PasswordHash, u.Honorific, u.FirstName, u.LastName, u.Phone,
		u.Role, u.EnterpriseID, u.IsActive, u.IsDeleted, u.EmailVerified,
		u.EmailVerifiedAt, u.FailedLoginAttempts, u.LockedUntil,
		u.PasswordChangedAt, u.MustChangePassword,
	)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE veritas_users SET is_deleted = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
