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
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"passwordHash"`

	Honorific *string `db:"honorific" json:"honorific,omitempty"`
	FirstName *string `db:"first_name" json:"firstName,omitempty"`
	LastName  *string `db:"last_name" json:"lastName,omitempty"`
	Phone     *string `db:"phone" json:"phone,omitempty"`

	Role         Role       `db:"role" json:"role"`
	EnterpriseID *uuid.UUID `db:"enterprise_id" json:"enterpriseId,omitempty"`

	IsActive  bool `db:"is_active" json:"isActive"`
	IsDeleted bool `db:"is_deleted" json:"isDeleted"`

	EmailVerified   bool       `db:"email_verified" json:"emailVerified"`
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"emailVerifiedAt,omitempty"`

	FailedLoginAttempts int        `db:"failed_login_attempts" json:"failedLoginAttempts"`
	LockedUntil         *time.Time `db:"locked_until" json:"lockedUntil,omitempty"`

	PasswordChangedAt  time.Time `db:"password_changed_at" json:"passwordChangedAt"`
	MustChangePassword bool      `db:"must_change_password" json:"mustChangePassword"`

	LastLoginAt   *time.Time `db:"last_login_at" json:"lastLoginAt,omitempty"`
	LastLoginIP   *string    `db:"last_login_ip" json:"lastLoginIp,omitempty"`
	LastUserAgent *string    `db:"last_user_agent" json:"lastUserAgent,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// NewUser provides a standard way to initialize a User with default values.
func NewUser(id uuid.UUID, email, passwordHash string, role Role) *User {
	now := time.Now().UTC()
	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,

		Role: role,

		IsActive:  true,
		IsDeleted: false,

		EmailVerified: false,

		PasswordChangedAt:  now,
		MustChangePassword: true,

		CreatedAt: now,
		UpdatedAt: now,
	}
}
