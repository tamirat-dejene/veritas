package domain

import (
	"context"
)

// MaintenanceUseCase defines background maintenance operations for the
// exam service, such as auto-publishing scheduled exams and closing expired ones.
type MaintenanceUseCase interface {
	PublishScheduledExams(ctx context.Context) error
	CloseExpiredExams(ctx context.Context) error
	ArchiveStaleExams(ctx context.Context) error
}
