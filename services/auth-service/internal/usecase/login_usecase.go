package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// LoginInput is the request payload for the login use case.
type LoginInput struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

// LoginOutput is the response payload for the login use case.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// LoginUseCase orchestrates user authentication with security auditing.
type LoginUseCase struct {
	pool             *pgxpool.Pool
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       domain.TokenService
	refreshService   domain.TokenService
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	eventPublisher   domain.EventPublisher
	log              *zap.Logger
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	eventPublisher domain.EventPublisher,
	log *zap.Logger,
) *LoginUseCase {
	return &LoginUseCase{
		pool:             pool,
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		refreshService:   refreshService,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		eventPublisher:   eventPublisher,
		log:              log,
	}
}

// Execute authenticates the user and returns a token pair on success.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. Find user by email.
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			uc.log.Warn("login attempt for unknown email", zap.String("email", input.Email))
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("LoginUseCase.Execute: FindByEmail: %w", err)
	}

	// 2. Reject if soft-deleted or inactive.
	if user.IsDeleted {
		return nil, domain.ErrUserDeleted
	}
	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	// 3. Check Account Lock.
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		uc.log.Warn("login attempt for locked account", zap.String("userId", user.ID.String()), zap.Time("lockedUntil", *user.LockedUntil))
		return nil, domain.ErrAccountLocked
	}

	// 4. Reject roles not served by this service.
	if _, ok := domain.AllowedAuthRoles[user.Role]; !ok {
		return nil, domain.ErrRoleNotPermitted
	}

	// 5. Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		uc.log.Warn("invalid password attempt", zap.String("userId", user.ID.String()))

		// Handle Login Failure (increment count, potentially lock)
		var lockUntil *time.Time
		if user.FailedLoginAttempts+1 >= 5 {
			l := time.Now().Add(15 * time.Minute)
			lockUntil = &l
			uc.log.Info("account locked due to too many failures", zap.String("userId", user.ID.String()))
		}

		if errUpdate := uc.userRepo.UpdateLoginFailure(ctx, user.ID, lockUntil); errUpdate != nil {
			uc.log.Error("failed to update login failure stats", zap.Error(errUpdate))
		}

		return nil, domain.ErrInvalidCredentials
	}

	// 6. Generate access token (JWT).
	accessToken, err := uc.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateAccessToken: %w", err)
	}

	// 7. Generate refresh token hash.
	rawRefreshToken, tokenHash, err := uc.refreshService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateRefreshToken: %w", err)
	}

	// 8 & 9 are combined into a transaction for ATOMICITY.
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// 8. Persist refresh token hash.
		rt := &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(uc.refreshTokenTTL).UTC(),
			Revoked:   false,
			CreatedAt: time.Now().UTC(),
		}
		if err := uc.refreshTokenRepo.WithTx(tx).Create(ctx, rt); err != nil {
			return fmt.Errorf("create refresh token: %w", err)
		}

		// 9. Update Login Audit Stats.
		if err := uc.userRepo.WithTx(tx).UpdateLoginSuccess(ctx, user.ID, input.IP, input.UserAgent); err != nil {
			return fmt.Errorf("update login success stats: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute transaction: %w", err)
	}

	uc.log.Info("user logged in successfully", zap.String("userId", user.ID.String()))

	// 10. Publish Login Event (Fire and Forget or handle as secondary)
	go func() {
		if err := uc.eventPublisher.PublishLogin(context.Background(), user.ID, user.Email); err != nil {
			uc.log.Error("failed to publish login event", zap.Error(err))
		}
	}()

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}
