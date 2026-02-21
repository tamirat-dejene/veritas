package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
)

// userRepository implements domain.UserRepository backed by PostgreSQL.
type userRepository struct {
	db postgres.PostgresClient
}

// NewUserRepository creates a new UserRepository.
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

// FindByEmail retrieves a user by email from veritas_users.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_users WHERE email = $1 LIMIT 1", userFields)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRow(ctx, query, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}
	return u, nil
}

// FindByID retrieves a user by ID from veritas_users.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_users WHERE id = $1 LIMIT 1", userFields)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRow(ctx, query, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return u, nil
}

// UpdateLoginSuccess updates last login fields and resets failed attempts.
func (r *userRepository) UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	const query = `
		UPDATE veritas_users
		SET last_login_at = NOW(),
		    last_login_ip = $2,
		    last_user_agent = $3,
		    failed_login_attempts = 0,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.Exec(ctx, query, userID, ip, userAgent)
	if err != nil {
		return fmt.Errorf("userRepository.UpdateLoginSuccess: %w", err)
	}
	return nil
}

// UpdateLoginFailure increments failed attempts and potentially sets a lock.
func (r *userRepository) UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time) error {
	const query = `
		UPDATE veritas_users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = $2,
		    updated_at = NOW()
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.Exec(ctx, query, userID, lockUntil)
	if err != nil {
		return fmt.Errorf("userRepository.UpdateLoginFailure: %w", err)
	}
	return nil
}
