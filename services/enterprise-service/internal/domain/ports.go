package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ─── Repository Interfaces ───────────────────────────────────────────────────

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	// Extended
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, page, limit int) ([]*User, int, error)
	FindByEnterpriseAndID(ctx context.Context, enterpriseID, userID uuid.UUID) (*User, error)
	CountByEnterprise(ctx context.Context, enterpriseID uuid.UUID) (int, error)
	UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
	UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error
	WithTx(tx pgx.Tx) UserRepository
}

type EnterpriseRepository interface {
	Create(ctx context.Context, enterprise *Enterprise) error
	FindByID(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	FindBySlug(ctx context.Context, slug string) (*Enterprise, error)
	Update(ctx context.Context, enterprise *Enterprise) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter map[string]interface{}) ([]*Enterprise, error)
	// Extended
	ListPaginated(ctx context.Context, filter EnterpriseFilter) ([]*Enterprise, int, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
	WithTx(tx pgx.Tx) EnterpriseRepository
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, page, limit int) ([]*AuditLog, int, error)
	WithTx(tx pgx.Tx) AuditRepository
}

// ─── Usecase Interfaces ──────────────────────────────────────────────────────

type EnterpriseUsecase interface {
	// Existing
	RegisterEnterprise(ctx context.Context, enterprise *Enterprise, owner *User) (*Enterprise, error)
	ApproveEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	SuspendEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	DeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	GetEnterprise(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise, adminID uuid.UUID) error

	// Discovery & Listing
	ListEnterprises(ctx context.Context, filter EnterpriseFilter) ([]*Enterprise, int, error)
	GetEnterpriseBySlug(ctx context.Context, slug string) (*Enterprise, error)
	GetMyEnterprise(ctx context.Context, enterpriseID uuid.UUID) (*Enterprise, error)

	// Branding & Settings
	UpdateBranding(ctx context.Context, id uuid.UUID, req UpdateBrandingRequest, adminID uuid.UUID) error
	UpdateSettings(ctx context.Context, id uuid.UUID, patch map[string]interface{}, adminID uuid.UUID) error

	// Subscription
	UpdateSubscription(ctx context.Context, id uuid.UUID, req UpdateSubscriptionRequest, adminID uuid.UUID) error
	CancelSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	RenewSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	GetSubscriptionInfo(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	SuspendForPayment(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error

	// Lifecycle & Governance
	ReactivateEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	RestoreEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	HardDeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error

	// Status, Domain, Audit
	GetEnterpriseStatus(ctx context.Context, id uuid.UUID) (*EnterpriseStatusResponse, error)
	ValidateCustomDomain(ctx context.Context, id uuid.UUID, adminID uuid.UUID) (*DomainValidationResult, error)
	GetEnterpriseSummary(ctx context.Context, id uuid.UUID) (*EnterpriseSummary, error)
	GetAuditLogs(ctx context.Context, id uuid.UUID, page, limit int) ([]*AuditLog, int, error)
}

type UserUsecase interface {
	CreateEnterpriseUser(ctx context.Context, enterpriseID uuid.UUID, req CreateUserRequest, adminID uuid.UUID) (*User, error)
	ListEnterpriseUsers(ctx context.Context, enterpriseID uuid.UUID, page, limit int) ([]*User, int, error)
	GetEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID) (*User, error)
	UpdateEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID, req UpdateUserRequest, adminID uuid.UUID) error
	DeactivateEnterpriseUser(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) error
	ResetUserPassword(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) (string, error)
	RecordLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
	RecordLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}
