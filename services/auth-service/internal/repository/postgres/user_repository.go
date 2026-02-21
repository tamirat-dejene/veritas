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

// FindByEmail retrieves a user by email address.
// Returns domain.ErrUserNotFound when no row matches.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT
			id, email, password_hash, role, enterprise_id,
			is_active, is_deleted, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRow(ctx, query, email)

	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.EnterpriseID,
		&u.IsActive,
		&u.IsDeleted,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}

	return &u, nil
}

// FindByID retrieves a user by primary key.
// Returns domain.ErrUserNotFound when no row matches.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const query = `
		SELECT
			id, email, password_hash, role, enterprise_id,
			is_active, is_deleted, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRow(ctx, query, id)

	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.EnterpriseID,
		&u.IsActive,
		&u.IsDeleted,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}

	return &u, nil
}
