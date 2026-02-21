package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
)

// findUserByID is a local helper that loads a user by ID using the UserRepository.
// We factor this out so all use-cases can share the same lookup logic.
func findUserByID(ctx context.Context, repo domain.UserRepository, id uuid.UUID) (*domain.User, error) {
	user, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("findUserByID: %w", err)
	}
	return user, nil
}
