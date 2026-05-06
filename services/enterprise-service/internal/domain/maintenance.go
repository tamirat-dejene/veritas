package domain

import (
	"context"
)

// MaintenanceUseCase defines background maintenance operations for the
// enterprise service, such as purging expired tokens and cleaning up soft-deleted records.
type MaintenanceUseCase interface {
	PurgeExpiredPasswordResetTokens(ctx context.Context) error
	HardDeleteExpiredEnterprises(ctx context.Context) error
	ResetExpiredAccountLocks(ctx context.Context) error
	PurgeOldAuditLogs(ctx context.Context) error
}
