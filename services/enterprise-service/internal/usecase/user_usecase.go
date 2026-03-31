package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// userUsecase implements domain.UserUsecase.
type userUsecase struct {
	pool           *pgxpool.Pool
	userRepo       domain.UserRepository
	enterpriseRepo domain.EnterpriseRepository
	auditRepo      domain.AuditRepository
}

// NewUserUsecase creates a UserUsecase.
func NewUserUsecase(
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	enterpriseRepo domain.EnterpriseRepository,
	auditRepo domain.AuditRepository,
) domain.UserUsecase {
	return &userUsecase{
		pool:           pool,
		userRepo:       userRepo,
		enterpriseRepo: enterpriseRepo,
		auditRepo:      auditRepo,
	}
}

// allowedEnterpriseRoles are the roles that an EnterpriseAdmin is permitted to create.
var allowedEnterpriseRoles = map[domain.Role]bool{
	domain.RoleEnterpriseAdmin: true,
	domain.RoleEnterpriseStaff: true,
	domain.RoleEnterpriseAuto:  true,
}

func (uc *userUsecase) emitUser(ctx context.Context, tx pgx.Tx, enterpriseID, actorID uuid.UUID, role string, event domain.AuditEvent, meta map[string]interface{}) {
	if meta == nil {
		meta = map[string]interface{}{}
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

func (uc *userUsecase) CreateEnterpriseUser(ctx context.Context, enterpriseID uuid.UUID, req domain.CreateUserRequest, adminID uuid.UUID) (*domain.User, error) {
	// Validate enterprise exists
	if _, err := uc.enterpriseRepo.FindByID(ctx, enterpriseID); err != nil {
		return nil, err
	}

	// Validate role
	if !allowedEnterpriseRoles[req.Role] {
		return nil, domain.ErrInvalidRole
	}

	// Check email uniqueness
	if existing, _ := uc.userRepo.FindByEmail(ctx, req.Email); existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Use NewUser constructor
	user := domain.NewUser(uuid.New(), req.Email, string(hash), req.Role)
	user.EnterpriseID = &enterpriseID
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Phone = req.Phone
	user.Honorific = req.Honorific
	user.MustChangePassword = true // force password change on first login

	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Create(ctx, user); err != nil {
			return err
		}

		uc.emitUser(ctx, tx, enterpriseID, adminID, string(domain.RoleEnterpriseAdmin), domain.EventUserCreated,
			map[string]interface{}{"user_id": user.ID.String(), "email": user.Email, "role": string(user.Role)})
		return nil
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *userUsecase) ListEnterpriseUsers(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.User, int, error) {
	if _, err := uc.enterpriseRepo.FindByID(ctx, enterpriseID); err != nil {
		return nil, 0, err
	}
	return uc.userRepo.ListByEnterprise(ctx, enterpriseID, params)
}

func (uc *userUsecase) GetEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID) (*domain.User, error) {
	return uc.userRepo.FindByEnterpriseAndID(ctx, enterpriseID, userID)
}

func (uc *userUsecase) UpdateEnterpriseUser(ctx context.Context, enterpriseID, userID uuid.UUID, req domain.UpdateUserRequest, adminID uuid.UUID) error {
	u, err := uc.userRepo.FindByEnterpriseAndID(ctx, enterpriseID, userID)
	if err != nil {
		return err
	}

	if req.FirstName != nil {
		u.FirstName = req.FirstName
	}
	if req.LastName != nil {
		u.LastName = req.LastName
	}
	if req.Phone != nil {
		u.Phone = req.Phone
	}
	if req.Honorific != nil {
		u.Honorific = req.Honorific
	}
	if req.Role != nil {
		if !allowedEnterpriseRoles[*req.Role] {
			return domain.ErrInvalidRole
		}
		u.Role = *req.Role
	}
	u.UpdatedAt = time.Now()

	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Update(ctx, u); err != nil {
			return err
		}
		uc.emitUser(ctx, tx, enterpriseID, adminID, string(domain.RoleEnterpriseAdmin), domain.EventUserUpdated,
			map[string]interface{}{"user_id": userID.String()})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (uc *userUsecase) DeactivateEnterpriseUser(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) error {
	u, err := uc.userRepo.FindByEnterpriseAndID(ctx, enterpriseID, userID)
	if err != nil {
		return err
	}
	u.IsActive = false
	u.UpdatedAt = time.Now()
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Update(ctx, u); err != nil {
			return err
		}
		uc.emitUser(ctx, tx, enterpriseID, adminID, string(domain.RoleEnterpriseAdmin), domain.EventUserDeactivated,
			map[string]interface{}{"user_id": userID.String()})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (uc *userUsecase) ResetUserPassword(ctx context.Context, enterpriseID, userID, adminID uuid.UUID) (string, error) {
	u, err := uc.userRepo.FindByEnterpriseAndID(ctx, enterpriseID, userID)
	if err != nil {
		return "", err
	}

	// Generate a temporary password
	tempPassword := fmt.Sprintf("Tmp-%s", uuid.New().String()[:8])
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash temp password: %w", err)
	}

	now := time.Now()
	u.PasswordHash = string(hash)
	u.PasswordChangedAt = now
	u.MustChangePassword = true
	u.UpdatedAt = now

	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Update(ctx, u); err != nil {
			return err
		}

		uc.emitUser(ctx, tx, enterpriseID, adminID, string(domain.RoleEnterpriseAdmin), domain.EventUserPasswordReset,
			map[string]interface{}{"user_id": userID.String()})
		return nil
	}); err != nil {
		return "", err
	}

	return tempPassword, nil
}

func (uc *userUsecase) RecordLoginSuccess(ctx context.Context, userID uuid.UUID, ip, userAgent string) error {
	return uc.userRepo.UpdateLoginSuccess(ctx, userID, ip, userAgent)
}

func (uc *userUsecase) RecordLoginFailure(ctx context.Context, userID uuid.UUID, lockUntil *time.Time, failedLoginAttempts int) error {
	return uc.userRepo.UpdateLoginFailure(ctx, userID, lockUntil, failedLoginAttempts)
}

func (uc *userUsecase) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return uc.userRepo.FindByEmail(ctx, email)
}

func (uc *userUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.userRepo.FindByID(ctx, id)
}
