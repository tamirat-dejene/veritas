package usecase

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type enterpriseUsecase struct {
	userRepo       domain.UserRepository
	enterpriseRepo domain.EnterpriseRepository
	auditRepo      domain.AuditRepository
}

func NewEnterpriseUsecase(
	userRepo domain.UserRepository,
	enterpriseRepo domain.EnterpriseRepository,
	auditRepo domain.AuditRepository,
) domain.EnterpriseUsecase {
	return &enterpriseUsecase{
		userRepo:       userRepo,
		enterpriseRepo: enterpriseRepo,
		auditRepo:      auditRepo,
	}
}

// ─── audit helper ────────────────────────────────────────────────────────────

func (uc *enterpriseUsecase) emit(ctx context.Context, enterpriseID, actorID uuid.UUID, role string, event domain.AuditEvent, meta map[string]interface{}) {
	if meta == nil {
		meta = map[string]interface{}{}
	}
	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		EnterpriseID: enterpriseID,
		ActorID:      actorID,
		ActorRole:    role,
		Event:        event,
		Metadata:     meta,
		CreatedAt:    time.Now(),
	})
}

func (uc *enterpriseUsecase) RegisterEnterprise(ctx context.Context, e *domain.Enterprise, owner *domain.User) (*domain.Enterprise, error) {
	zap.L().Info("Registering new enterprise", zap.String("slug", e.Slug), zap.String("owner_email", owner.Email))

	// 1. Check if slug exists
	existing, err := uc.enterpriseRepo.FindBySlug(ctx, e.Slug)
	if err != nil && err != domain.ErrEnterpriseNotFound {
		zap.L().Error("Failed to check if slug exists", zap.Error(err), zap.String("slug", e.Slug))
		return nil, err
	}
	if existing != nil {
		zap.L().Warn("Enterprise slug already exists", zap.String("slug", e.Slug))
		return nil, domain.ErrSlugAlreadyExists
	}

	// 2. Check if owner email exists
	existingUser, err := uc.userRepo.FindByEmail(ctx, owner.Email)
	if err != nil && err != domain.ErrUserNotFound {
		zap.L().Error("Failed to check if owner email exists", zap.Error(err), zap.String("email", owner.Email))
		return nil, err
	}
	if existingUser != nil {
		zap.L().Warn("Owner email already exists", zap.String("email", owner.Email))
		return nil, domain.ErrEmailAlreadyExists
	}

	// 3. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(owner.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		zap.L().Error("Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	owner.PasswordHash = string(hashedPassword)

	// 4. Create Owner User
	owner.ID = uuid.New()
	owner.Role = domain.RoleEnterpriseAdmin
	owner.IsActive = true
	owner.CreatedAt = time.Now()
	owner.UpdatedAt = owner.CreatedAt
	owner.PasswordChangedAt = owner.CreatedAt

	if err := uc.userRepo.Create(ctx, owner); err != nil {
		zap.L().Error("Failed to create owner user", zap.Error(err), zap.String("email", owner.Email))
		return nil, fmt.Errorf("failed to create owner user: %w", err)
	}
	zap.L().Info("Owner user created", zap.String("user_id", owner.ID.String()))

	// 5. Create Enterprise
	e.ID = uuid.New()
	if e.Settings == nil {
		e.Settings = make(map[string]interface{})
	}
	e.OwnerAccountID = owner.ID
	e.Status = domain.StatusPendingApproval
	e.CreatedAt = time.Now()
	e.UpdatedAt = e.CreatedAt
	e.CreatedBy = owner.ID
	e.UpdatedBy = owner.ID

	if err := uc.enterpriseRepo.Create(ctx, e); err != nil {
		zap.L().Error("Failed to create enterprise", zap.Error(err), zap.String("slug", e.Slug))
		return nil, fmt.Errorf("failed to create enterprise: %w", err)
	}
	zap.L().Info("Enterprise created", zap.String("enterprise_id", e.ID.String()))

	// 6. Update Owner's EnterpriseID
	owner.EnterpriseID = &e.ID
	if err := uc.userRepo.Update(ctx, owner); err != nil {
		zap.L().Error("Failed to update owner with enterprise ID", zap.Error(err), zap.String("user_id", owner.ID.String()))
		return nil, fmt.Errorf("failed to update owner with enterprise ID: %w", err)
	}
	zap.L().Info("Owner updated with enterprise ID", zap.String("user_id", owner.ID.String()))

	return e, nil
}

func (uc *enterpriseUsecase) ApproveEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	e.Status = domain.StatusActive
	e.ApprovedAt = &now
	e.UpdatedAt = now
	e.UpdatedBy = adminID

	return uc.enterpriseRepo.Update(ctx, e)
}

func (uc *enterpriseUsecase) SuspendEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	e.Status = domain.StatusSuspended
	e.SuspendedAt = &now
	e.UpdatedAt = now
	e.UpdatedBy = adminID

	return uc.enterpriseRepo.Update(ctx, e)
}

func (uc *enterpriseUsecase) DeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	if err := uc.enterpriseRepo.Delete(ctx, id); err != nil {
		return err
	}
	uc.emit(ctx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseDeleted, nil)
	return nil
}

func (uc *enterpriseUsecase) GetEnterprise(ctx context.Context, id uuid.UUID) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindByID(ctx, id)
}

func (uc *enterpriseUsecase) UpdateEnterprise(ctx context.Context, e *domain.Enterprise, adminID uuid.UUID) error {
	existing, err := uc.enterpriseRepo.FindByID(ctx, e.ID)
	if err != nil {
		return err
	}

	// Update only allowed fields
	existing.DisplayName = e.DisplayName
	existing.LegalName = e.LegalName
	existing.ContactEmail = e.ContactEmail
	existing.LogoURL = e.LogoURL
	existing.PrimaryColor = e.PrimaryColor
	existing.SecondaryColor = e.SecondaryColor
	existing.CustomDomain = e.CustomDomain
	existing.ContactPhone = e.ContactPhone
	existing.AddressLine1 = e.AddressLine1
	existing.AddressLine2 = e.AddressLine2
	existing.City = e.City
	existing.Country = e.Country
	existing.Settings = e.Settings
	existing.UpdatedAt = time.Now()
	existing.UpdatedBy = adminID

	return uc.enterpriseRepo.Update(ctx, existing)
}

// ─── Discovery & Listing ─────────────────────────────────────────────────────

func (uc *enterpriseUsecase) ListEnterprises(ctx context.Context, filter domain.EnterpriseFilter) ([]*domain.Enterprise, int, error) {
	return uc.enterpriseRepo.ListPaginated(ctx, filter)
}

func (uc *enterpriseUsecase) GetEnterpriseBySlug(ctx context.Context, slug string) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindBySlug(ctx, slug)
}

func (uc *enterpriseUsecase) GetMyEnterprise(ctx context.Context, enterpriseID uuid.UUID) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindByID(ctx, enterpriseID)
}

// ─── Branding & Settings ─────────────────────────────────────────────────────

func (uc *enterpriseUsecase) UpdateBranding(ctx context.Context, id uuid.UUID, req domain.UpdateBrandingRequest, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if req.LogoURL != nil {
		e.LogoURL = req.LogoURL
	}
	if req.PrimaryColor != nil {
		e.PrimaryColor = req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		e.SecondaryColor = req.SecondaryColor
	}
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	uc.emit(ctx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventBrandingUpdated, nil)
	return nil
}

func (uc *enterpriseUsecase) UpdateSettings(ctx context.Context, id uuid.UUID, patch map[string]interface{}, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	// JSON merge patch: overlay patch onto existing settings
	if e.Settings == nil {
		e.Settings = map[string]interface{}{}
	}
	for k, v := range patch {
		e.Settings[k] = v
	}
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	uc.emit(ctx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventSettingsUpdated, nil)
	return nil
}

// ─── Lifecycle & Governance ──────────────────────────────────────────────────

func (uc *enterpriseUsecase) ReactivateEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Status != domain.StatusSuspended {
		return domain.ErrInvalidStatus
	}
	now := time.Now()
	e.Status = domain.StatusActive
	e.SuspendedAt = nil
	e.UpdatedAt = now
	e.UpdatedBy = adminID
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	uc.emit(ctx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseReactivated, nil)
	return nil
}

func (uc *enterpriseUsecase) RestoreEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Status != domain.StatusDeleted {
		return domain.ErrInvalidStatus
	}
	if e.RetentionUntil != nil && time.Now().After(*e.RetentionUntil) {
		return domain.ErrRetentionActive
	}
	now := time.Now()
	e.Status = domain.StatusActive
	e.DeletedAt = nil
	e.RetentionUntil = nil
	e.UpdatedAt = now
	e.UpdatedBy = adminID
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	uc.emit(ctx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseRestored, nil)
	return nil
}

func (uc *enterpriseUsecase) HardDeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Status != domain.StatusDeleted {
		return domain.ErrInvalidStatus
	}
	// Only allow if retention period has expired
	if e.RetentionUntil != nil && time.Now().Before(*e.RetentionUntil) {
		return domain.ErrRetentionActive
	}
	uc.emit(ctx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseHardDeleted, nil)
	return uc.enterpriseRepo.HardDelete(ctx, id)
}

// ─── Status, Domain, Audit ───────────────────────────────────────────────────

func (uc *enterpriseUsecase) GetEnterpriseStatus(ctx context.Context, id uuid.UUID) (*domain.EnterpriseStatusResponse, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.EnterpriseStatusResponse{
		EnterpriseID:       e.ID,
		Status:             e.Status,
		SubscriptionStatus: e.SubscriptionStatus,
		ApprovedAt:         e.ApprovedAt,
		SuspendedAt:        e.SuspendedAt,
		DeletedAt:          e.DeletedAt,
		RetentionUntil:     e.RetentionUntil,
		CurrentPeriodEnd:   e.CurrentPeriodEnd,
	}, nil
}

func (uc *enterpriseUsecase) ValidateCustomDomain(ctx context.Context, id uuid.UUID, adminID uuid.UUID) (*domain.DomainValidationResult, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.CustomDomain == nil || *e.CustomDomain == "" {
		return nil, domain.ErrDomainValidation
	}

	result := &domain.DomainValidationResult{
		Domain: *e.CustomDomain,
	}

	// Check TXT record
	txtRecords, _ := net.LookupTXT(*e.CustomDomain)
	expected := fmt.Sprintf("veritas-verify=%s", e.ID.String())
	for _, txt := range txtRecords {
		if strings.Contains(txt, expected) {
			result.TXTRecordFound = true
			break
		}
	}

	// Check CNAME
	cname, _ := net.LookupCNAME(*e.CustomDomain)
	if strings.Contains(cname, "veritas") {
		result.CNAMEFound = true
	}

	result.Valid = result.TXTRecordFound || result.CNAMEFound
	if result.Valid {
		result.Details = "domain validation succeeded"
		uc.emit(ctx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventDomainValidated, map[string]interface{}{"domain": *e.CustomDomain})
	} else {
		result.Details = fmt.Sprintf("expected TXT record '%s' or CNAME pointing to veritas", expected)
	}
	return result, nil
}

func (uc *enterpriseUsecase) GetEnterpriseSummary(ctx context.Context, id uuid.UUID) (*domain.EnterpriseSummary, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	userCount, err := uc.userRepo.CountByEnterprise(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.EnterpriseSummary{
		EnterpriseID:       e.ID,
		DisplayName:        e.DisplayName,
		Status:             e.Status,
		SubscriptionStatus: e.SubscriptionStatus,
		SubscriptionExpiry: e.CurrentPeriodEnd,
		UserCount:          userCount,
		ActiveExamCount:    -1, // requires exam-service client
		ActiveSessionCount: -1, // requires candidate-service client
	}, nil
}

func (uc *enterpriseUsecase) GetAuditLogs(ctx context.Context, id uuid.UUID, page, limit int) ([]*domain.AuditLog, int, error) {
	// Confirm enterprise exists first
	if _, err := uc.enterpriseRepo.FindByID(ctx, id); err != nil {
		return nil, 0, err
	}
	return uc.auditRepo.ListByEnterprise(ctx, id, page, limit)
}
