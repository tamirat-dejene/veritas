package domain

import (
	"context"

	"github.com/google/uuid"
)

type ConsistencyUseCase interface {
	HandleEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID) error
}
