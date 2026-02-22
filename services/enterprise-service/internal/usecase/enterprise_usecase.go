package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type enterpriseUsecase struct {
	userRepo       domain.UserRepository
	enterpriseRepo domain.EnterpriseRepository
}

func NewEnterpriseUsecase(userRepo domain.UserRepository, enterpriseRepo domain.EnterpriseRepository) domain.EnterpriseUsecase {
	return &enterpriseUsecase{
		userRepo:       userRepo,
		enterpriseRepo: enterpriseRepo,
	}
}

func (uc *enterpriseUsecase) RegisterEnterprise(ctx context.Context, e *domain.Enterprise, owner *domain.User) (*domain.Enterprise, error) {
	// 1. Check if slug exists
	existing, _ := uc.enterpriseRepo.FindBySlug(ctx, e.Slug)
	if existing != nil {
		return nil, domain.ErrSlugAlreadyExists
	}

	// 2. Check if owner email exists
	existingUser, _ := uc.userRepo.FindByEmail(ctx, owner.Email)
	if existingUser != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// 3. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(owner.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
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
		return nil, fmt.Errorf("failed to create owner user: %w", err)
	}

	// 5. Create Enterprise
	e.ID = uuid.New()
	e.OwnerAccountID = owner.ID
	e.Status = domain.StatusPendingApproval
	e.CreatedAt = time.Now()
	e.UpdatedAt = e.CreatedAt
	e.CreatedBy = owner.ID // Initially created by owner
	e.UpdatedBy = owner.ID

	if err := uc.enterpriseRepo.Create(ctx, e); err != nil {
		// Cleanup owner if enterprise creation fails? (In a real app, use a transaction)
		return nil, fmt.Errorf("failed to create enterprise: %w", err)
	}

	// 6. Update Owner's EnterpriseID
	owner.EnterpriseID = &e.ID
	if err := uc.userRepo.Update(ctx, owner); err != nil {
		return nil, fmt.Errorf("failed to update owner with enterprise ID: %w", err)
	}

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
	return uc.enterpriseRepo.Delete(ctx, id)
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
