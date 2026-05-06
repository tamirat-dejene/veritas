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
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
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
	enterpriseServiceClient  domain.EnterpriseServiceClient
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
	enterpriseServiceClient domain.EnterpriseServiceClient,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	log *zap.Logger,
) *RefreshUseCase {
	return &RefreshUseCase{
		pool:             pool,
		enterpriseServiceClient:  enterpriseServiceClient,
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
		l := logger.WithContext(ctx, uc.log)
		// 2. Lock and load token row.
		rt, err := uc.refreshTokenRepo.WithTx(tx).FindByHashForUpdate(ctx, tokenHash)
		if err != nil {
			if err == domain.ErrTokenNotFound {
				l.Warn("refresh attempt with unknown token", zap.String("token_hash", tokenHash))
				return domain.ErrTokenNotFound
			}
			return fmt.Errorf("find token for update: %w", err)
		}

		// 3. Reject if already revoked.
		if rt.Revoked {
			l.Warn("refresh attempt with revoked token", zap.String("tokenId", rt.ID.String()))
			return domain.ErrTokenRevoked
		}

		// 4. Reject if expired.
		if time.Now().UTC().After(rt.ExpiresAt) {
			l.Warn("refresh attempt with expired token", zap.String("tokenId", rt.ID.String()))
			return domain.ErrTokenExpired
		}

		// 5. Load associated user by ID.
		user, err := uc.enterpriseServiceClient.FindByID(ctx, rt.UserID)
		if err != nil {
			if err == domain.ErrUserNotFound {
				return domain.ErrInvalidCredentials
			}
			return fmt.Errorf("find user by id: %w", err)
		}

		// 6. Reject if user is deleted or inactive.
		if user.IsDeleted {
			l.Warn("refresh attempt for deleted user", zap.String("userId", user.ID.String()))
			return domain.ErrUserDeleted
		}
		if !user.IsActive {
			l.Warn("refresh attempt for inactive user", zap.String("userId", user.ID.String()))
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
		logger.WithContext(ctx, uc.log).Error("refresh transaction failed", zap.Error(err))
		return nil, fmt.Errorf("RefreshUseCase.Execute transaction: %w", err)
	}

	logger.WithContext(ctx, uc.log).Info("token refreshed successfully", zap.String("userId", userID.String()))

	return &RefreshOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}

// PurgeExpiredTokens removes tokens that are past their expiration date.
func (uc *RefreshUseCase) PurgeExpiredTokens(ctx context.Context) error {
	l := logger.WithContext(ctx, uc.log)
	
	count, err := uc.refreshTokenRepo.DeleteExpired(ctx, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("purge expired tokens: %w", err)
	}

	if count > 0 {
		l.Info("purged expired refresh tokens", zap.Int64("count", count))
	}
	
	return nil
}

// AuditSessionIntegrity checks for users with an excessive number of active sessions.
func (uc *RefreshUseCase) AuditSessionIntegrity(ctx context.Context) error {
	l := logger.WithContext(ctx, uc.log)
	const threshold = 50

	userIDs, err := uc.refreshTokenRepo.FindUsersWithExcessiveSessions(ctx, threshold)
	if err != nil {
		return fmt.Errorf("audit session integrity: %w", err)
	}

	if len(userIDs) > 0 {
		l.Warn("found users with excessive active sessions", 
			zap.Int("user_count", len(userIDs)),
			zap.Int("threshold", threshold),
			zap.Any("user_ids", userIDs),
		)
	}

	return nil
}

