package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role defines the role of a user in the system.
type Role string

const (
	RoleSystemAdmin     Role = "SystemAdmin"
	RoleEnterpriseAdmin Role = "EnterpriseAdmin"
	RoleEnterpriseAuto  Role = "EnterpriseAuto"
	RoleEnterpriseStaff Role = "EnterpriseStaff"
	RoleExamCandidate   Role = "ExamCandidate"
)

// AllowedAuthRoles lists the roles that are permitted to authenticate via this service.
var AllowedAuthRoles = map[Role]struct{}{
	RoleSystemAdmin:     {},
	RoleEnterpriseAdmin: {},
	RoleEnterpriseAuto:  {},
	RoleEnterpriseStaff: {},
}

// User represents a row in the veritas_users table.
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
