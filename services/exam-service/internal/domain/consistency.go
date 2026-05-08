package domain

import (
	"context"

	"github.com/google/uuid"
)

// ConsistencyUseCase is the port for asynchronous data consistency operations
type ConsistencyUseCase interface {
	HandleEnterpriseSuspended(ctx context.Context, enterpriseID uuid.UUID) error
	HandleEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID) error
}
