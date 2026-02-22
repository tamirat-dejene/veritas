package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type EnterpriseRepository interface {
	Create(ctx context.Context, enterprise *Enterprise) error
	FindByID(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	FindBySlug(ctx context.Context, slug string) (*Enterprise, error)
	Update(ctx context.Context, enterprise *Enterprise) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter map[string]interface{}) ([]*Enterprise, error)
}

type EnterpriseUsecase interface {
	RegisterEnterprise(ctx context.Context, enterprise *Enterprise, owner *User) (*Enterprise, error)
	ApproveEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	SuspendEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	DeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	GetEnterprise(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise, adminID uuid.UUID) error
}
