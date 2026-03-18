package usecase

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/token"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

// LogoutInput is the request payload for the logout use case.
type LogoutInput struct {
	RefreshToken string
}

// LogoutUseCase revokes the provided refresh token.
type LogoutUseCase struct {
	pool             *pgxpool.Pool
	refreshTokenRepo domain.RefreshTokenRepository
	log              *zap.Logger
}

// NewLogoutUseCase creates a new LogoutUseCase.
func NewLogoutUseCase(
	pool *pgxpool.Pool,
	refreshTokenRepo domain.RefreshTokenRepository,
	log *zap.Logger,
) *LogoutUseCase {
	return &LogoutUseCase{
		pool:             pool,
		refreshTokenRepo: refreshTokenRepo,
		log:              log,
	}
}

// Execute revokes the refresh token associated with the provided raw token.
// This operation is idempotent: revoking an already-revoked token is not an error.
func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	// 1. Hash the incoming raw token to look it up.
	tokenHash := token.HashToken(input.RefreshToken)

	var tokenID string

	// 2-4. Lock token row and revoke inside one transaction.
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		l := logger.WithContext(ctx, uc.log)
		rt, err := uc.refreshTokenRepo.WithTx(tx).FindByHashForUpdate(ctx, tokenHash)
		if err != nil {
			if err == domain.ErrTokenNotFound {
				l.Warn("logout attempt with unknown token (no-op)")
				return nil
			}
			return fmt.Errorf("find token for update: %w", err)
		}

		tokenID = rt.ID.String()

		if rt.Revoked {
			l.Info("logout called on already-revoked token (no-op)", zap.String("tokenId", rt.ID.String()))
			return nil
		}

		if err := uc.refreshTokenRepo.WithTx(tx).Revoke(ctx, rt.ID); err != nil {
			if err == domain.ErrTokenRevoked {
				return nil
			}
			return fmt.Errorf("revoke: %w", err)
		}
		return nil
	}); err != nil {
		logger.WithContext(ctx, uc.log).Error("logout transaction failed", zap.Error(err))
		return fmt.Errorf("LogoutUseCase.Execute transaction: %w", err)
	}

	if tokenID != "" {
		logger.WithContext(ctx, uc.log).Info("user logged out successfully", zap.String("tokenId", tokenID))
	}
	return nil
}
