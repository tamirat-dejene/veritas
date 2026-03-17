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
	ID           uuid.UUID 
	Email        string    
	PasswordHash string    

	Honorific *string 
	FirstName *string 
	LastName  *string 
	Phone     *string 

	Role         Role       
	EnterpriseID *uuid.UUID 

	IsActive  bool 
	IsDeleted bool 

	EmailVerified   bool       
	EmailVerifiedAt *time.Time 

	FailedLoginAttempts int        
	LockedUntil         *time.Time 

	PasswordChangedAt  time.Time 
	MustChangePassword bool      

	LastLoginAt   *time.Time 
	LastLoginIP   *string    
	LastUserAgent *string    

	CreatedAt time.Time 
	UpdatedAt time.Time 
}
