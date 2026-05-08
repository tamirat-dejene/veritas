package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

type consistencyUseCase struct {
	refreshTokenRepo domain.RefreshTokenRepository
	enterpriseClient domain.EnterpriseServiceClient
	log              *zap.Logger
}

// NewConsistencyUseCase creates a new instance of ConsistencyUseCase.
func NewConsistencyUseCase(
	refreshTokenRepo domain.RefreshTokenRepository,
	enterpriseClient domain.EnterpriseServiceClient,
	log *zap.Logger,
) domain.ConsistencyUseCase {
	return &consistencyUseCase{
		refreshTokenRepo: refreshTokenRepo,
		enterpriseClient: enterpriseClient,
		log:              log,
	}
}

// RevokeUserSessions revokes all sessions (refresh tokens) for a given user.
// This is triggered when a user is deactivated, deleted, or their password is changed/reset.
func (uc *consistencyUseCase) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	l := logger.WithContext(ctx, uc.log).With(zap.String("user_id", userID.String()))
	
	if err := uc.refreshTokenRepo.DeleteAllForUser(ctx, userID); err != nil {
		l.Error("failed to revoke user sessions", zap.Error(err))
		return fmt.Errorf("ConsistencyUseCase.RevokeUserSessions: %w", err)
	}

	l.Info("successfully revoked all sessions for user")
	return nil
}

// RevokeEnterpriseSessions revokes all sessions for all users under a given enterprise.
// This is triggered when an enterprise is suspended or deleted.
func (uc *consistencyUseCase) RevokeEnterpriseSessions(ctx context.Context, enterpriseID uuid.UUID) error {
	l := logger.WithContext(ctx, uc.log).With(zap.String("enterprise_id", enterpriseID.String()))

	userIDs, err := uc.enterpriseClient.ListUsersByEnterprise(ctx, enterpriseID)
	if err != nil {
		l.Error("failed to fetch enterprise users", zap.Error(err))
		return fmt.Errorf("ConsistencyUseCase.RevokeEnterpriseSessions: fetch users: %w", err)
	}

	var errorsCount int
	for _, userID := range userIDs {
		if err := uc.refreshTokenRepo.DeleteAllForUser(ctx, userID); err != nil {
			l.Warn("failed to revoke sessions for enterprise user", zap.String("user_id", userID.String()), zap.Error(err))
			errorsCount++
		}
	}

	if errorsCount > 0 {
		return fmt.Errorf("ConsistencyUseCase.RevokeEnterpriseSessions: %d errors occurred while revoking sessions", errorsCount)
	}

	l.Info("successfully revoked all sessions for enterprise", zap.Int("users_count", len(userIDs)))
	return nil
}
