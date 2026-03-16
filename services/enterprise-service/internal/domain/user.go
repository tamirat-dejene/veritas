package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleSystemAdmin     Role = "SystemAdmin"
	RoleEnterpriseAdmin Role = "EnterpriseAdmin"
	RoleEnterpriseAuto  Role = "EnterpriseAuto"
	RoleEnterpriseStaff Role = "EnterpriseStaff"
	RoleExamCandidate   Role = "ExamCandidate"
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`

	Honorific *string `db:"honorific"`
	FirstName *string `db:"first_name"`
	LastName  *string `db:"last_name"`
	Phone     *string `db:"phone"`

	Role         Role       `db:"role"`
	EnterpriseID *uuid.UUID `db:"enterprise_id"`

	IsActive  bool `db:"is_active"`
	IsDeleted bool `db:"is_deleted"`

	EmailVerified   bool       `db:"email_verified"`
	EmailVerifiedAt *time.Time `db:"email_verified_at"`

	FailedLoginAttempts int        `db:"failed_login_attempts"`
	LockedUntil         *time.Time `db:"locked_until"`

	PasswordChangedAt  time.Time `db:"password_changed_at"`
	MustChangePassword bool      `db:"must_change_password"`

	LastLoginAt   *time.Time `db:"last_login_at"`
	LastLoginIP   *string    `db:"last_login_ip"`
	LastUserAgent *string    `db:"last_user_agent"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// NewUser provides a standard way to initialize a User with default values.
func NewUser(id uuid.UUID, email, passwordHash string, role Role) *User {
	now := time.Now().UTC()
	return &User{
		ID:                 id,
		Email:              email,
		PasswordHash:       passwordHash,

		Role:               role,

		IsActive:           true,
		IsDeleted:          false,

		EmailVerified:      false,

		PasswordChangedAt:  now,
		MustChangePassword: true,

		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
