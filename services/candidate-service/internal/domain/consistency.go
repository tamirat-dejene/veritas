package domain

import (
	"context"

	"github.com/google/uuid"
)

// ConsistencyUseCase is the port for asynchronous data consistency operations driven by external events.
type ConsistencyUseCase interface {
	HandleEnterpriseDeactivated(ctx context.Context, enterpriseID uuid.UUID) error
	HandleExamClosed(ctx context.Context, examID uuid.UUID) error
}
