package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type AuthService interface {
	Register(ctx context.Context, email, password, role, firstName, lastName string) (*User, error)
	Login(ctx context.Context, email, password string) (string, error)
	Validate(ctx context.Context, token string) (*User, error)
}
