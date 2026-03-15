package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type subscriptionUsecase struct {
	enterpriseRepo domain.EnterpriseRepository
	auditRepo      domain.AuditRepository
}

// SubscriptionUsecase is embedded into EnterpriseUsecase; see subscription methods on enterpriseUsecase below.
// This file adds the remaining methods to satisfy the EnterpriseUsecase interface.

// Patch enterpriseUsecase to also embed subscription logic via the same struct.
// The methods below are defined directly on enterpriseUsecase (same package).

func (uc *enterpriseUsecase) UpdateSubscription(ctx context.Context, id uuid.UUID, req domain.UpdateSubscriptionRequest, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if req.SubscriptionPlanID != nil {
		e.SubscriptionPlanID = req.SubscriptionPlanID
	}
	if req.SubscriptionStatus != nil {
		e.SubscriptionStatus = req.SubscriptionStatus
	}
	if req.PeriodStart != nil {
		e.CurrentPeriodStart = req.PeriodStart
	}
	if req.PeriodEnd != nil {
		e.CurrentPeriodEnd = req.PeriodEnd
	}
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID

	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventSubscriptionUpdated,
			map[string]interface{}{"subscription_status": fmt.Sprintf("%v", req.SubscriptionStatus)})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (uc *enterpriseUsecase) CancelSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.SubscriptionStatus == nil {
		return domain.ErrSubscriptionRequired
	}
	canceled := domain.SubCanceled
	e.SubscriptionStatus = &canceled
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventSubscriptionCanceled, nil)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (uc *enterpriseUsecase) RenewSubscription(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.SubscriptionStatus == nil {
		return domain.ErrSubscriptionRequired
	}
	active := domain.SubActive
	e.SubscriptionStatus = &active
	// Extend period by 1 year from current end, or from now if already expired
	base := time.Now()
	if e.CurrentPeriodEnd != nil && e.CurrentPeriodEnd.After(base) {
		base = *e.CurrentPeriodEnd
	}
	newEnd := base.AddDate(1, 0, 0)
	e.CurrentPeriodEnd = &newEnd
	e.UpdatedAt = time.Now()
	e.UpdatedBy = adminID
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, adminID, string(domain.RoleSystemAdmin), domain.EventSubscriptionRenewed,
			map[string]interface{}{"new_period_end": newEnd.Format(time.RFC3339)})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (uc *enterpriseUsecase) GetSubscriptionInfo(ctx context.Context, id uuid.UUID) (*domain.Enterprise, error) {
	return uc.enterpriseRepo.FindByID(ctx, id)
}

func (uc *enterpriseUsecase) SuspendForPayment(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	e, err := uc.enterpriseRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if e.Status == domain.StatusSuspended {
		return domain.ErrInvalidStatus
	}
	now := time.Now()
	e.Status = domain.StatusSuspended
	e.SuspendedAt = &now
	pastDue := domain.SubPastDue
	e.SubscriptionStatus = &pastDue
	e.UpdatedAt = now
	e.UpdatedBy = actorID
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		if err := uc.enterpriseRepo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		uc.emit(ctx, tx, id, actorID, string(domain.RoleSystemAdmin), domain.EventSubscriptionSuspended, nil)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// Suppress the unused import if subscriptionUsecase struct is not instantiated directly.
var _ = subscriptionUsecase{}
