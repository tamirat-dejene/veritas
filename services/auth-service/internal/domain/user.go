package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleStudent    Role = "student"
	RoleInstructor Role = "instructor"
	RoleProctor    Role = "proctor"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         Role      `json:"role" db:"role"`
	FirstName    string    `json:"firstName" db:"first_name"`
	LastName     string    `json:"lastName" db:"last_name"`
	EnterpriseID *string   `json:"enterpriseId,omitempty" db:"enterprise_id"`
	Tier         string    `json:"tier,omitempty" db:"tier"` // e.g., "free", "basic", "premium"
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}
