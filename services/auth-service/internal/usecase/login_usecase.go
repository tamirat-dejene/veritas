package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// LoginInput is the request payload for the login use case.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput is the response payload for the login use case.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// LoginUseCase orchestrates user authentication.
type LoginUseCase struct {
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       domain.TokenService
	refreshService   domain.TokenService
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	log              *zap.Logger
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	log *zap.Logger,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		refreshService:   refreshService,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		log:              log,
	}
}

// Execute authenticates the user and returns a token pair on success.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. Find user by email.
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		// Do not distinguish "not found" from "bad password" to the caller.
		if err == domain.ErrUserNotFound {
			uc.log.Warn("login attempt for unknown email", zap.String("email", input.Email))
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("LoginUseCase.Execute: FindByEmail: %w", err)
	}

	// 2. Reject if soft-deleted.
	if user.IsDeleted {
		uc.log.Warn("login attempt for deleted account", zap.String("userId", user.ID.String()))
		return nil, domain.ErrUserDeleted
	}

	// 3. Reject if inactive.
	if !user.IsActive {
		uc.log.Warn("login attempt for inactive account", zap.String("userId", user.ID.String()))
		return nil, domain.ErrUserInactive
	}

	// 4. Reject roles not served by this service.
	if _, ok := domain.AllowedAuthRoles[user.Role]; !ok {
		uc.log.Warn("login attempt from disallowed role",
			zap.String("userId", user.ID.String()),
			zap.String("role", string(user.Role)),
		)
		return nil, domain.ErrRoleNotPermitted
	}

	// 5. Verify password using constant-time bcrypt comparison.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		uc.log.Warn("invalid password attempt", zap.String("userId", user.ID.String()))
		return nil, domain.ErrInvalidCredentials
	}

	// 6. Generate access token (JWT).
	accessToken, err := uc.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateAccessToken: %w", err)
	}

	// 7. Generate raw refresh token and its SHA-256 hash.
	rawRefreshToken, tokenHash, err := uc.refreshService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateRefreshToken: %w", err)
	}

	// 8. Persist the hash (NEVER the raw token).
	now := time.Now().UTC()
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		Revoked:   false,
		CreatedAt: now,
	}
	if err := uc.refreshTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: Create refresh token: %w", err)
	}

	uc.log.Info("user logged in successfully", zap.String("userId", user.ID.String()))

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}
