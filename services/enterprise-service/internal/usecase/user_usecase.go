package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	pool             *pgxpool.Pool
	userRepo         domain.UserRepository
	enterpriseRepo   domain.EnterpriseRepository
	auditRepo        domain.AuditRepository
	eventPublisher   domain.EventPublisher
	passwordResetRepo domain.PasswordResetRepository
	frontendBaseURL  string
}

// NewUserUsecase creates a UserUsecase.
func NewUserUsecase(
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	enterpriseRepo domain.EnterpriseRepository,
	auditRepo domain.AuditRepository,
	eventPublisher domain.EventPublisher,
	passwordResetRepo domain.PasswordResetRepository,
	frontendBaseURL string,
) domain.UserUsecase {
	return &userUsecase{
		pool:              pool,
		userRepo:          userRepo,
		enterpriseRepo:    enterpriseRepo,
		auditRepo:         auditRepo,
		eventPublisher:    eventPublisher,
		passwordResetRepo: passwordResetRepo,
		frontendBaseURL:   frontendBaseURL,
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
	ent, err := uc.enterpriseRepo.FindByID(ctx, enterpriseID)
	if err != nil {
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

	// Publish event after transaction commits
	name := ""
	if user.FirstName != nil && user.LastName != nil {
		name = fmt.Sprintf("%s %s", *user.FirstName, *user.LastName)
	} else if user.FirstName != nil {
		name = *user.FirstName
	} else {
		name = "Staff Member"
	}

	if uc.eventPublisher != nil && (user.Role == domain.RoleEnterpriseStaff || user.Role == domain.RoleEnterpriseAdmin) {
		_ = uc.eventPublisher.PublishEnterpriseStaffCreated(context.Background(), user.ID, user.Email, name, req.Password, ent.LegalName)
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

func (uc *userUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, req domain.ChangePasswordRequest) error {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	now := time.Now()
	u.PasswordHash = string(hash)
	u.PasswordChangedAt = now
	u.MustChangePassword = false
	u.UpdatedAt = now

	enterpriseID := uuid.Nil
	if u.EnterpriseID != nil {
		enterpriseID = *u.EnterpriseID
	}

	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Update(ctx, u); err != nil {
			return err
		}

		uc.emitUser(ctx, tx, enterpriseID, userID, string(u.Role), domain.EventUserPasswordChanged,
			map[string]interface{}{"user_id": userID.String()})
		return nil
	}); err != nil {
		return err
	}

	return nil
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

// ForgotPassword initiates a self-service password reset. It is intentionally
// silent — the same 200 OK is returned whether the email exists or not to
// prevent user enumeration.
func (uc *userUsecase) ForgotPassword(ctx context.Context, email string) error {
	// Silently resolve the user; any error (not found, inactive) is swallowed.
	u, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil || !u.IsActive || u.IsDeleted {
		return nil
	}

	// Invalidate all previous tokens for this user.
	if err := uc.passwordResetRepo.InvalidatePreviousTokens(ctx, u.ID); err != nil {
		return fmt.Errorf("forgot_password: invalidate previous tokens: %w", err)
	}

	// Generate a 32-byte cryptographically-secure random token.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return fmt.Errorf("forgot_password: generate token: %w", err)
	}
	rawToken := hex.EncodeToString(rawBytes) // 64-char hex string

	// Store only the SHA-256 hash — never the raw token.
	hashBytes := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hashBytes[:])

	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	if err := uc.passwordResetRepo.CreateToken(ctx, u.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("forgot_password: create token: %w", err)
	}

	// Build the reset link using the raw (un-hashed) token.
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", uc.frontendBaseURL, rawToken)

	// Resolve a display name for the email greeting.
	name := "User"
	if u.FirstName != nil && *u.FirstName != "" {
		name = *u.FirstName
		if u.LastName != nil && *u.LastName != "" {
			name += " " + *u.LastName
		}
	}

	// Publish the event asynchronously — a Kafka failure must not block the response.
	if uc.eventPublisher != nil {
		_ = uc.eventPublisher.PublishPasswordResetRequested(context.Background(), u.ID, u.Email, name, resetLink)
	}

	// Emit an audit log entry (best-effort).
	enterpriseID := uuid.Nil
	if u.EnterpriseID != nil {
		enterpriseID = *u.EnterpriseID
	}
	uc.emitUser(ctx, nil, enterpriseID, u.ID, string(u.Role), domain.EventUserForgotPassword,
		map[string]interface{}{"user_id": u.ID.String()})

	return nil
}

// ResetPasswordViaToken validates the one-time token and sets a new password
// for the user in a single atomic transaction.
func (uc *userUsecase) ResetPasswordViaToken(ctx context.Context, req domain.ResetPasswordRequest) error {
	// Hash the supplied raw token to look it up in the DB.
	hashBytes := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hashBytes[:])

	token, err := uc.passwordResetRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		// FindByTokenHash returns ErrResetTokenInvalid for no-rows.
		return domain.ErrResetTokenInvalid
	}

	// Guard against already-used tokens.
	if token.Used {
		return domain.ErrResetTokenUsed
	}

	// Guard against expired tokens.
	if time.Now().UTC().After(token.ExpiresAt) {
		return domain.ErrResetTokenInvalid
	}

	// Fetch the owning user.
	u, err := uc.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return err
	}

	// Hash the new password.
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("reset_password_via_token: hash password: %w", err)
	}

	now := time.Now().UTC()
	u.PasswordHash = string(hash)
	u.PasswordChangedAt = now
	u.MustChangePassword = false
	u.UpdatedAt = now

	enterpriseID := uuid.Nil
	if u.EnterpriseID != nil {
		enterpriseID = *u.EnterpriseID
	}

	// Update user + mark token used in one transaction.
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.userRepo.WithTx(tx).Update(ctx, u); err != nil {
			return err
		}
		if err := uc.passwordResetRepo.WithTx(tx).MarkUsed(ctx, token.ID); err != nil {
			return err
		}
		uc.emitUser(ctx, tx, enterpriseID, u.ID, string(u.Role), domain.EventUserPasswordResetViaToken,
			map[string]interface{}{"user_id": u.ID.String()})
		return nil
	})
}
