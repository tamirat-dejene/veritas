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
// Candidate authentication is handled by a separate service.
var AllowedAuthRoles = map[Role]struct{}{
	RoleSystemAdmin:     {},
	RoleEnterpriseAdmin: {},
	RoleEnterpriseAuto:  {},
	RoleEnterpriseStaff: {},
}

// User represents a row in the users table.
type User struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Role         Role       `db:"role"`
	EnterpriseID *uuid.UUID `db:"enterprise_id"`
	IsActive     bool       `db:"is_active"`
	IsDeleted    bool       `db:"is_deleted"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}
