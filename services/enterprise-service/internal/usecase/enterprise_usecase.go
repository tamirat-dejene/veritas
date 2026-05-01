package usecase

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type enterpriseUsecase struct {
	pool           *pgxpool.Pool
	userRepo       domain.UserRepository
	enterpriseRepo domain.EnterpriseRepository
	auditRepo      domain.AuditRepository
	eventPublisher domain.EventPublisher
	paymentClient  domain.PaymentClient
	examClient     domain.ExamClient
	candidateClient domain.CandidateClient
}

func NewEnterpriseUsecase(
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	enterpriseRepo domain.EnterpriseRepository,
	auditRepo domain.AuditRepository,
	eventPublisher domain.EventPublisher,
	paymentClient domain.PaymentClient,
	examClient domain.ExamClient,
	candidateClient domain.CandidateClient,
) domain.EnterpriseUsecase {
	return &enterpriseUsecase{
		pool:           pool,
		userRepo:       userRepo,
		enterpriseRepo: enterpriseRepo,
		auditRepo:      auditRepo,
		eventPublisher: eventPublisher,
		paymentClient:  paymentClient,
		examClient:     examClient,
		candidateClient: candidateClient,
	}
}

// ─── audit helper ────────────────────────────────────────────────────────────

func (uc *enterpriseUsecase) emit(ctx context.Context, tx pgx.Tx, enterpriseID, actorID uuid.UUID, role string, event domain.AuditEvent, meta map[string]interface{}) {
	if meta == nil {
		meta = map[string]any{}
	}
	repo := uc.auditRepo
	if tx != nil {
		repo = repo.WithTx(tx)
	}
	_ = repo.Create(ctx, &domain.AuditLog{
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
	// 1. Check if slug exists
	existing, err := uc.enterpriseRepo.FindBySlug(ctx, e.Slug, uuid.Nil)
	if err != nil && err != domain.ErrEnterpriseNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrSlugAlreadyExists
	}

	// 2. Check if owner email exists
	existingUser, err := uc.userRepo.FindByEmail(ctx, owner.Email)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, err
	}
	if existingUser != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// 3. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(owner.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser := domain.NewUser(uuid.New(), owner.Email, string(hashedPassword), domain.RoleEnterpriseAdmin)
	newUser.MustChangePassword = false

	newEnterprise := domain.NewEnterprise(uuid.New(), e.Slug, e.DisplayName, newUser.ID)
	newEnterprise.LegalName = e.LegalName
	newEnterprise.ContactEmail = e.ContactEmail
	newEnterprise.CreatedBy = newUser.ID
	newEnterprise.UpdatedBy = newUser.ID

	if err = RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		txEnterpriseRepo := uc.enterpriseRepo.WithTx(tx)
		txUserRepo := uc.userRepo.WithTx(tx)
		txAuditRepo := uc.auditRepo.WithTx(tx)

		if err := txUserRepo.Create(ctx, newUser); err != nil {
			return fmt.Errorf("failed to create owner user: %w", err)
		}
		if err := txEnterpriseRepo.Create(ctx, newEnterprise); err != nil {
			return fmt.Errorf("failed to create enterprise: %w", err)
		}
		newUser.EnterpriseID = &newEnterprise.ID
		if err := txUserRepo.Update(ctx, newUser); err != nil {
			return fmt.Errorf("failed to update owner with enterprise ID: %w", err)
		}
		auditLog := &domain.AuditLog{
			ID:           uuid.New(),
			EnterpriseID: newEnterprise.ID,
			ActorID:      newUser.ID,
			ActorRole:    string(newUser.Role),
			Event:        domain.EventEnterpriseCreated,
			CreatedAt:    time.Now(),
			Metadata:     map[string]any{"slug": newEnterprise.Slug},
		}
		if err := txAuditRepo.Create(ctx, auditLog); err != nil {
			return fmt.Errorf("failed to create audit log: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := uc.eventPublisher.PublishEnterpriseCreated(ctx, newEnterprise.ID, newEnterprise.LegalName, owner.Email); err != nil {
		zap.L().Error("failed to publish enterprise.created event", zap.Error(err))
	}

	return newEnterprise, nil
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
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	_ = uc.eventPublisher.PublishEnterpriseApproved(ctx, e.ID, e.LegalName, e.ContactEmail)
	return nil
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
	if err := uc.enterpriseRepo.Update(ctx, e); err != nil {
		return err
	}
	_ = uc.eventPublisher.PublishEnterpriseSuspended(ctx, e.ID, e.LegalName, e.ContactEmail, "administrative suspension")
	return nil
}

func (uc *enterpriseUsecase) DeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Delete(ctx, id); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseDeleted, nil)
		_ = uc.eventPublisher.PublishEnterpriseDeleted(ctx, id, e.LegalName, e.ContactEmail)
		return nil
	})
}

func (uc *enterpriseUsecase) GetEnterprise(ctx context.Context, id uuid.UUID) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindByID(ctx, id)
}

func (uc *enterpriseUsecase) UpdateEnterprise(ctx context.Context, e *domain.Enterprise, adminID uuid.UUID) error {
	existing, err := uc.enterpriseRepo.FindByID(ctx, e.ID)
	if err != nil {
		return err
	}
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

	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, existing); err != nil {
			return err
		}
		uc.emit(ctx, tx, existing.ID, adminID, string(domain.RoleEnterpriseAdmin), domain.EventEnterpriseUpdated, nil)
		return nil
	})
}

// ─── Discovery & Listing ─────────────────────────────────────────────────────

func (uc *enterpriseUsecase) ListEnterprises(ctx context.Context, filter domain.EnterpriseFilter) ([]*domain.Enterprise, int, error) {
	return uc.enterpriseRepo.ListPaginated(ctx, filter)
}

func (uc *enterpriseUsecase) GetEnterpriseBySlug(ctx context.Context, slug string, adminID uuid.UUID) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindBySlug(ctx, slug, adminID)
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
	log.Printf("Updating branding for enterprise: %v", e)
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
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventBrandingUpdated, nil)
		return nil
	})
}

func (uc *enterpriseUsecase) UpdateSettings(ctx context.Context, id uuid.UUID, patch map[string]interface{}, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Settings == nil {
		e.Settings = map[string]interface{}{}
	}
	for k, v := range patch {
		e.Settings[k] = v
	}
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventSettingsUpdated, nil)
		return nil
	})
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
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseReactivated, nil)
		_ = uc.eventPublisher.PublishEnterpriseReactivated(ctx, e.ID, e.LegalName, e.ContactEmail)
		return nil
	})
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
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseRestored, nil)
		_ = uc.eventPublisher.PublishEnterpriseRestored(ctx, e.ID, e.LegalName, e.ContactEmail)
		return nil
	})
}

func (uc *enterpriseUsecase) HardDeleteEnterprise(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Status != domain.StatusDeleted {
		return domain.ErrInvalidStatus
	}
	if e.RetentionUntil != nil && time.Now().Before(*e.RetentionUntil) {
		return domain.ErrRetentionActive
	}
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventEnterpriseHardDeleted, nil)
		if err := uc.enterpriseRepo.WithTx(tx).HardDelete(ctx, id); err != nil {
			return err
		}
		_ = uc.eventPublisher.PublishEnterpriseHardDeleted(ctx, e.ID, e.LegalName, e.ContactEmail)
		return nil
	})
}

// ─── Status, Domain, Audit ───────────────────────────────────────────────────

// GetEnterpriseStatus fetches the enterprise's lifecycle status and enriches
// it with live subscription data from the payment-service.
func (uc *enterpriseUsecase) GetEnterpriseStatus(ctx context.Context, id uuid.UUID) (*domain.EnterpriseStatusResponse, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &domain.EnterpriseStatusResponse{
		EnterpriseID:   e.ID,
		Status:         e.Status,
		ApprovedAt:     e.ApprovedAt,
		SuspendedAt:    e.SuspendedAt,
		DeletedAt:      e.DeletedAt,
		RetentionUntil: e.RetentionUntil,
	}

	// Enrich with live subscription data — non-fatal if payment-service is unavailable.
	sub, err := uc.paymentClient.GetActiveSubscription(ctx, id)
	if err != nil {
		zap.L().Warn("GetEnterpriseStatus: failed to fetch subscription from payment-service",
			zap.String("enterprise_id", id.String()), zap.Error(err))
	} else {
		resp.Subscription = sub
	}

	return resp, nil
}

func (uc *enterpriseUsecase) ValidateCustomDomain(ctx context.Context, id uuid.UUID, adminID uuid.UUID) (*domain.DomainValidationResult, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.CustomDomain == nil || *e.CustomDomain == "" {
		return nil, domain.ErrDomainValidation
	}

	result := &domain.DomainValidationResult{Domain: *e.CustomDomain}

	txtRecords, _ := net.LookupTXT(*e.CustomDomain)
	expected := fmt.Sprintf("veritas-verify=%s", e.ID.String())
	for _, txt := range txtRecords {
		if strings.Contains(txt, expected) {
			result.TXTRecordFound = true
			break
		}
	}

	cname, _ := net.LookupCNAME(*e.CustomDomain)
	if strings.Contains(cname, "veritas") {
		result.CNAMEFound = true
	}

	result.Valid = result.TXTRecordFound || result.CNAMEFound
	if result.Valid {
		result.Details = "domain validation succeeded"
		if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
			uc.emit(ctx, tx, id, adminID, string(domain.RoleEnterpriseAdmin), domain.EventDomainValidated, map[string]interface{}{"domain": *e.CustomDomain})
			return nil
		}); err != nil {
			return nil, err
		}
	} else {
		result.Details = fmt.Sprintf("expected TXT record '%s' or CNAME pointing to veritas", expected)
	}
	return result, nil
}

// GetEnterpriseSummary returns an overview of the enterprise enriched with
// live subscription data from the payment-service.
func (uc *enterpriseUsecase) GetEnterpriseSummary(ctx context.Context, id uuid.UUID) (*domain.EnterpriseSummary, error) {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	userCount, err := uc.userRepo.CountByEnterprise(ctx, id)
	if err != nil {
		return nil, err
	}

	summary := &domain.EnterpriseSummary{
		EnterpriseID:       e.ID,
		DisplayName:        e.DisplayName,
		Status:             e.Status,
		UserCount:          userCount,
		ActiveExamCount:    0,
		ActiveSessionCount: 0,
	}

	if uc.examClient != nil {
		if count, err := uc.examClient.GetActiveExamsCount(ctx, id); err == nil {
			summary.ActiveExamCount = count
		}
	}
	if uc.candidateClient != nil {
		if count, err := uc.candidateClient.GetActiveSessionsCount(ctx, id); err == nil {
			summary.ActiveSessionCount = count
		}
	}

	sub, err := uc.paymentClient.GetActiveSubscription(ctx, id)
	if err != nil {
		zap.L().Warn("GetEnterpriseSummary: failed to fetch subscription from payment-service",
			zap.String("enterprise_id", id.String()), zap.Error(err))
	} else {
		summary.Subscription = sub
	}

	return summary, nil
}

func (uc *enterpriseUsecase) GetAuditLogs(ctx context.Context, id uuid.UUID, params pagination.Params) ([]*domain.AuditLog, int, error) {
	if _, err := uc.enterpriseRepo.FindByID(ctx, id); err != nil {
		return nil, 0, err
	}
	return uc.auditRepo.ListByEnterprise(ctx, id, params)
}

func (uc *enterpriseUsecase) SuspendForPayment(ctx context.Context, enterpriseID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, enterpriseID)
	if err != nil {
		return err
	}
	if e.Status == domain.StatusSuspended {
		return nil
	}
	now := time.Now()
	e.Status = domain.StatusSuspended
	e.SuspendedAt = &now
	e.UpdatedAt = now
	e.UpdatedBy = uuid.Nil
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, enterpriseID, uuid.Nil, "system", domain.EventSubscriptionSuspended, nil)
		_ = uc.eventPublisher.PublishEnterpriseSuspended(ctx, e.ID, e.LegalName, e.ContactEmail, "payment issue")
		return nil
	})
}
