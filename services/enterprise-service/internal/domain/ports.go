package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// ─── Repository Interfaces ───────────────────────────────────────────────────

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*User, int, error)
	FindByEnterpriseAndID(ctx context.Context, enterpriseID, userID uuid.UUID) (*User, error)
	CountByEnterprise(ctx context.Context, enterpriseID uuid.UUID) (int, error)
	UpdateLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
	UpdateLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error
	WithTx(tx pgx.Tx) UserRepository
}

type PasswordResetRepository interface {
	CreateToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	InvalidatePreviousTokens(ctx context.Context, userID uuid.UUID) error
	MarkUsed(ctx context.Context, tokenID uuid.UUID) error
	WithTx(tx pgx.Tx) PasswordResetRepository
}

type EnterpriseRepository interface {
	Create(ctx context.Context, enterprise *Enterprise) error
	FindByID(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	FindBySlug(ctx context.Context, slug string, adminID uuid.UUID) (*Enterprise, error)
	Update(ctx context.Context, enterprise *Enterprise) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter map[string]interface{}) ([]*Enterprise, error)
	ListPaginated(ctx context.Context, filter EnterpriseFilter) ([]*Enterprise, int, error)
	HardDelete(ctx context.Context, id uuid.UUID) error
	WithTx(tx pgx.Tx) EnterpriseRepository
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*AuditLog, int, error)
	WithTx(tx pgx.Tx) AuditRepository
}

// ─── Usecase Interfaces ──────────────────────────────────────────────────────

type EnterpriseUsecase interface {
	// Registration & core CRUD
	RegisterEnterprise(ctx context.Context, enterprise *Enterprise, owner *User) (*Enterprise, error)
	ApproveEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	SuspendEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	DeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	GetEnterprise(ctx context.Context, id uuid.UUID) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise, adminID uuid.UUID) error

	// Discovery & Listing
	ListEnterprises(ctx context.Context, filter EnterpriseFilter) ([]*Enterprise, int, error)
	GetEnterpriseBySlug(ctx context.Context, slug string, adminID uuid.UUID) (*Enterprise, error)
	GetMyEnterprise(ctx context.Context, enterpriseID uuid.UUID) (*Enterprise, error)

	// Branding & Settings
	UpdateBranding(ctx context.Context, id uuid.UUID, req UpdateBrandingRequest, adminID uuid.UUID) error
	UpdateSettings(ctx context.Context, id uuid.UUID, patch map[string]interface{}, adminID uuid.UUID) error

	// Lifecycle & Governance
	ReactivateEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	RestoreEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	HardDeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error
	SuspendForPayment(ctx context.Context, enterpriseID uuid.UUID) error

	// Status, Domain, Audit
	GetEnterpriseStatus(ctx context.Context, id uuid.UUID) (*EnterpriseStatusResponse, error)
	ValidateCustomDomain(ctx context.Context, id uuid.UUID, adminID uuid.UUID) (*DomainValidationResult, error)
	GetEnterpriseSummary(ctx context.Context, id uuid.UUID) (*EnterpriseSummary, error)
	GetAuditLogs(ctx context.Context, id uuid.UUID, params pagination.Params) ([]*AuditLog, int, error)
}

type UserUsecase interface {
	CreateEnterpriseUser(ctx context.Context, enterpriseID uuid.UUID, req CreateUserRequest, adminID uuid.UUID) (*User, error)
	ListEnterpriseUsers(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*User, int, error)
	GetEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID) (*User, error)
	UpdateEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID, req UpdateUserRequest, adminID uuid.UUID) error
	DeactivateEnterpriseUser(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) error
	ActivateEnterpriseUser(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) error
	DeleteEnterpriseUser(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) error
	ResetUserPassword(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) (string, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPasswordViaToken(ctx context.Context, req ResetPasswordRequest) error
	RecordLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
	RecordLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type EventPublisher interface {
	PublishEnterpriseCreated(ctx context.Context, enterpriseID uuid.UUID, legalName string, ownerEmail string) error
	PublishEnterpriseApproved(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error
	PublishEnterpriseSuspended(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail, reason string) error
	PublishEnterpriseDeleted(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error
	PublishEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error
	PublishEnterpriseReactivated(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error
	PublishEnterpriseRestored(ctx context.Context, enterpriseID uuid.UUID, legalName, contactEmail string) error
	PublishEnterpriseStaffCreated(ctx context.Context, staffID uuid.UUID, email, name, tempPassword, enterpriseName string) error
	PublishPasswordResetRequested(ctx context.Context, userID uuid.UUID, email, name, resetLink string) error
	PublishUserDeactivated(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error
	PublishUserActivated(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error
	PublishUserDeleted(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error
	PublishUserPasswordChanged(ctx context.Context, userID uuid.UUID, email, name, enterpriseName string) error
	PublishUserPasswordResetAdmin(ctx context.Context, userID uuid.UUID, email, name, tempPassword, enterpriseName string) error
}

type PaymentClient interface {
	// Used to fetch live subscription state for status/summary endpoints.
	GetActiveSubscription(ctx context.Context, enterpriseID uuid.UUID) (*SubscriptionSnapshot, error)
}

type ExamClient interface {
	GetActiveExamsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error)
}

type CandidateClient interface {
	GetActiveSessionsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error)
}