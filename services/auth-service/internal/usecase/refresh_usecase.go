package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/token"
	"go.uber.org/zap"
)

// RefreshInput is the request payload for the refresh use case.
type RefreshInput struct {
	RefreshToken string
}

// RefreshOutput is the response payload for the refresh use case.
type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// RefreshUseCase implements token rotation: revoke old token, issue new pair.
type RefreshUseCase struct {
	userRepo         domain.UserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       domain.TokenService
	refreshService   domain.TokenService
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	log              *zap.Logger
}

// NewRefreshUseCase creates a new RefreshUseCase.
func NewRefreshUseCase(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	log *zap.Logger,
) *RefreshUseCase {
	return &RefreshUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		refreshService:   refreshService,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		log:              log,
	}
}

// Execute validates the incoming refresh token and, if valid, issues a new token pair.
func (uc *RefreshUseCase) Execute(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	// 1. Hash the incoming raw token to look it up in the database.
	tokenHash := token.HashToken(input.RefreshToken)

	// 2. Look up the token by hash.
	rt, err := uc.refreshTokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		if err == domain.ErrTokenNotFound {
			uc.log.Warn("refresh attempt with unknown token")
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("RefreshUseCase.Execute: FindByHash: %w", err)
	}

	// 3. Reject if already revoked.
	if rt.Revoked {
		uc.log.Warn("refresh attempt with revoked token", zap.String("tokenId", rt.ID.String()))
		return nil, domain.ErrTokenRevoked
	}

	// 4. Reject if expired.
	if time.Now().UTC().After(rt.ExpiresAt) {
		uc.log.Warn("refresh attempt with expired token", zap.String("tokenId", rt.ID.String()))
		return nil, domain.ErrTokenExpired
	}

	// 5. Load associated user by ID.
	user, err := findUserByID(ctx, uc.userRepo, rt.UserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("RefreshUseCase.Execute: findUserByID: %w", err)
	}

	// 6. Reject if user is deleted or inactive.
	if user.IsDeleted {
		return nil, domain.ErrUserDeleted
	}
	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	// 7. Revoke the old refresh token (token rotation).
	if err := uc.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("RefreshUseCase.Execute: Revoke old token: %w", err)
	}

	// 8. Generate new access token.
	accessToken, err := uc.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("RefreshUseCase.Execute: GenerateAccessToken: %w", err)
	}

	// 9. Generate new refresh token.
	rawRefreshToken, newTokenHash, err := uc.refreshService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("RefreshUseCase.Execute: GenerateRefreshToken: %w", err)
	}

	// 10. Persist the new refresh token hash.
	now := time.Now().UTC()
	newRT := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		Revoked:   false,
		CreatedAt: now,
	}
	if err := uc.refreshTokenRepo.Create(ctx, newRT); err != nil {
		return nil, fmt.Errorf("RefreshUseCase.Execute: Create new refresh token: %w", err)
	}

	uc.log.Info("token refreshed successfully", zap.String("userId", user.ID.String()))

	return &RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}
