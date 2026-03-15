package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool             *pgxpool.Pool
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
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	log *zap.Logger,
) *RefreshUseCase {
	return &RefreshUseCase{
		pool:             pool,
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

	var (
		accessToken     string
		rawRefreshToken string
		newTokenHash    string
		userID          uuid.UUID
	)

	// 2-10 are combined into one transaction to make token rotation concurrency-safe.
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// 2. Lock and load token row.
		rt, err := uc.refreshTokenRepo.WithTx(tx).FindByHashForUpdate(ctx, tokenHash)
		if err != nil {
			if err == domain.ErrTokenNotFound {
				uc.log.Warn("refresh attempt with unknown token")
				return domain.ErrTokenNotFound
			}
			return fmt.Errorf("find token for update: %w", err)
		}

		// 3. Reject if already revoked.
		if rt.Revoked {
			uc.log.Warn("refresh attempt with revoked token", zap.String("tokenId", rt.ID.String()))
			return domain.ErrTokenRevoked
		}

		// 4. Reject if expired.
		if time.Now().UTC().After(rt.ExpiresAt) {
			uc.log.Warn("refresh attempt with expired token", zap.String("tokenId", rt.ID.String()))
			return domain.ErrTokenExpired
		}

		// 5. Load associated user by ID.
		user, err := findUserByID(ctx, uc.userRepo.WithTx(tx), rt.UserID)
		if err != nil {
			if err == domain.ErrUserNotFound {
				return domain.ErrInvalidCredentials
			}
			return fmt.Errorf("find user by id: %w", err)
		}

		// 6. Reject if user is deleted or inactive.
		if user.IsDeleted {
			return domain.ErrUserDeleted
		}
		if !user.IsActive {
			return domain.ErrUserInactive
		}

		// 7. Revoke old token while row is locked.
		if err := uc.refreshTokenRepo.WithTx(tx).Revoke(ctx, rt.ID); err != nil {
			return fmt.Errorf("revoke old token: %w", err)
		}

		// 8. Generate new access token.
		accessToken, err = uc.jwtService.GenerateAccessToken(user)
		if err != nil {
			return fmt.Errorf("generate access token: %w", err)
		}

		// 9. Generate new refresh token.
		rawRefreshToken, newTokenHash, err = uc.refreshService.GenerateRefreshToken()
		if err != nil {
			return fmt.Errorf("generate refresh token: %w", err)
		}

		// 10. Persist new refresh token hash.
		now := time.Now().UTC()
		newRT := &domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    user.ID,
			TokenHash: newTokenHash,
			ExpiresAt: now.Add(uc.refreshTokenTTL),
			Revoked:   false,
			CreatedAt: now,
		}
		if err := uc.refreshTokenRepo.WithTx(tx).Create(ctx, newRT); err != nil {
			return fmt.Errorf("create new refresh token: %w", err)
		}

		userID = user.ID
		return nil
	}); err != nil {
		return nil, fmt.Errorf("RefreshUseCase.Execute transaction: %w", err)
	}

	uc.log.Info("token refreshed successfully", zap.String("userId", userID.String()))

	return &RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}
