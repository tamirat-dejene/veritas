package domain

import (
	"context"
)

// MaintenanceUseCase handles automated background maintenance tasks
// for the candidate service, such as auto-submitting expired sessions
// and revoking expired enrollments.
type MaintenanceUseCase interface {
	ProcessExpiredSessions(ctx context.Context) error
	ProcessExpiredEnrollments(ctx context.Context) error
}
