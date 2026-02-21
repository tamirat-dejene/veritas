package usecase

import (
	"context"
	"fmt"

	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/infrastructure/token"
	"go.uber.org/zap"
)

// LogoutInput is the request payload for the logout use case.
type LogoutInput struct {
	RefreshToken string
}

// LogoutUseCase revokes the provided refresh token.
type LogoutUseCase struct {
	refreshTokenRepo domain.RefreshTokenRepository
	log              *zap.Logger
}

// NewLogoutUseCase creates a new LogoutUseCase.
func NewLogoutUseCase(
	refreshTokenRepo domain.RefreshTokenRepository,
	log *zap.Logger,
) *LogoutUseCase {
	return &LogoutUseCase{
		refreshTokenRepo: refreshTokenRepo,
		log:              log,
	}
}

// Execute revokes the refresh token associated with the provided raw token.
// This operation is idempotent: revoking an already-revoked token is not an error.
func (uc *LogoutUseCase) Execute(ctx context.Context, input LogoutInput) error {
	// 1. Hash the incoming raw token to look it up.
	tokenHash := token.HashToken(input.RefreshToken)

	// 2. Look up the token by hash.
	rt, err := uc.refreshTokenRepo.FindByHash(ctx, tokenHash)
	if err != nil {
		// If token is not found, treat as a no-op (already logged out or invalid).
		if err == domain.ErrTokenNotFound {
			uc.log.Warn("logout attempt with unknown token (no-op)")
			return nil
		}
		return fmt.Errorf("LogoutUseCase.Execute: FindByHash: %w", err)
	}

	// 3. If already revoked, treat as idempotent success.
	if rt.Revoked {
		uc.log.Info("logout called on already-revoked token (no-op)", zap.String("tokenId", rt.ID.String()))
		return nil
	}

	// 4. Revoke the token.
	if err := uc.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return fmt.Errorf("LogoutUseCase.Execute: Revoke: %w", err)
	}

	uc.log.Info("user logged out successfully", zap.String("tokenId", rt.ID.String()))
	return nil
}
