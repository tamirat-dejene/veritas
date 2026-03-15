package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) WithTx(tx pgx.Tx) domain.UserRepository {
	return &userRepository{db: tx}
}

const userFields = `
	id, email, password_hash, honorific, first_name, last_name, phone, role, enterprise_id,
	is_active, is_deleted, email_verified, email_verified_at, failed_login_attempts, locked_until,
	password_changed_at, must_change_password, last_login_at, last_login_ip, last_user_agent,
	created_at, updated_at
`

func scanUser(row pgx.Row) (*domain.User, error) {
	var m userModel
	err := row.Scan(
		&m.ID, &m.Email, &m.PasswordHash, &m.Honorific, &m.FirstName, &m.LastName, &m.Phone, &m.Role, &m.EnterpriseID,
		&m.IsActive, &m.IsDeleted, &m.EmailVerified, &m.EmailVerifiedAt, &m.FailedLoginAttempts, &m.LockedUntil,
		&m.PasswordChangedAt, &m.MustChangePassword, &m.LastLoginAt, &m.LastLoginIP, &m.LastUserAgent,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return m.toDomain(), nil
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

// ListByEnterprise returns paginated users for an enterprise.
func (r *userRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, page, limit int) ([]*domain.User, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	countQ := "SELECT COUNT(*) FROM veritas_users WHERE enterprise_id = $1 AND is_deleted = false"
	if err := r.db.QueryRow(ctx, countQ, enterpriseID).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataQ := fmt.Sprintf(
		"SELECT %s FROM veritas_users WHERE enterprise_id = $1 AND is_deleted = false ORDER BY created_at DESC LIMIT $2 OFFSET $3",
		userFields,
	)
	rows, err := r.db.Query(ctx, dataQ, enterpriseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// FindByEnterpriseAndID fetches a user scoped to an enterprise.
func (r *userRepository) FindByEnterpriseAndID(ctx context.Context, enterpriseID, userID uuid.UUID) (*domain.User, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM veritas_users WHERE enterprise_id = $1 AND id = $2 AND is_deleted = false LIMIT 1",
		userFields,
	)
	row := r.db.QueryRow(ctx, query, enterpriseID, userID)
	return scanUser(row)
}

// CountByEnterprise returns the number of active users for an enterprise.
func (r *userRepository) CountByEnterprise(ctx context.Context, enterpriseID uuid.UUID) (int, error) {
	var count int
	const q = "SELECT COUNT(*) FROM veritas_users WHERE enterprise_id = $1 AND is_deleted = false AND is_active = true"
	err := r.db.QueryRow(ctx, q, enterpriseID).Scan(&count)
	return count, err
}

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
	commandTag, err := r.db.Exec(ctx, query, userID, ip, userAgent)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error {
	const query = `
		UPDATE veritas_users
		SET failed_login_attempts = $3,
		    locked_until = $2,
		    updated_at = NOW()
		WHERE id = $1
	`
	var pgLockUntil pgtype.Timestamptz
	if lockUntil != nil {
		pgLockUntil = pgtype.Timestamptz{Time: *lockUntil, Valid: true}
	} else {
		pgLockUntil = pgtype.Timestamptz{Valid: false}
	}

	commandTag, err := r.db.Exec(ctx, query, userID, pgLockUntil, failedLoginAttempts)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
