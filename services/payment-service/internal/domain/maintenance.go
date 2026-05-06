package domain

import (
	"context"
)

// MaintenanceUseCase defines background maintenance operations for the
// payment service, such as expiring lapsed subscriptions and cleaning up stale data.
type MaintenanceUseCase interface {
	ExpireLapsedSubscriptions(ctx context.Context) error
	MarkOverdueInvoicesUncollectible(ctx context.Context) error
	PurgeOldWebhookEvents(ctx context.Context) error
}
