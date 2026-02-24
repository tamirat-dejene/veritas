package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	pg "github.com/tamirat-dejene/veritas/shared/db/pg"
)

const (
	planFields = "id, name, slug, description, price, currency, billing_cycle, features, is_active, created_at, updated_at"
	subFields  = "id, enterprise_id, plan_id, status, current_period_start, current_period_end, cancel_at_period_end, canceled_at, ended_at, stripe_customer_id, stripe_subscription_id, created_at, updated_at"
)

type subscriptionRepository struct {
	db pg.PostgresClient
}

func NewSubscriptionRepository(db pg.PostgresClient) domain.SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) ListPlans(ctx context.Context) ([]*domain.SubscriptionPlan, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans WHERE is_active = true", planFields)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*domain.SubscriptionPlan
	for rows.Next() {
		var p domain.SubscriptionPlan
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plans = append(plans, &p)
	}
	return plans, nil
}

func (r *subscriptionRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans WHERE id = $1", planFields)
	var p domain.SubscriptionPlan
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, domain.ErrPlanNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *subscriptionRepository) GetPlanBySlug(ctx context.Context, slug string) (*domain.SubscriptionPlan, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans WHERE slug = $1", planFields)
	var p domain.SubscriptionPlan
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, domain.ErrPlanNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *subscriptionRepository) GetSubscriptionByEnterpriseID(ctx context.Context, enterpriseID uuid.UUID) (*domain.EnterpriseSubscription, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_enterprise_subscriptions WHERE enterprise_id = $1", subFields)
	var s domain.EnterpriseSubscription
	err := r.db.QueryRow(ctx, query, enterpriseID).Scan(
		&s.ID, &s.EnterpriseID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd, &s.CanceledAt, &s.EndedAt, &s.StripeCustomerID, &s.StripeSubscriptionID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *subscriptionRepository) CreateSubscription(ctx context.Context, s *domain.EnterpriseSubscription) error {
	query := `
		INSERT INTO veritas_enterprise_subscriptions (
			id, enterprise_id, plan_id, status, current_period_start, current_period_end, 
			cancel_at_period_end, canceled_at, ended_at, stripe_customer_id, stripe_subscription_id, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		s.ID, s.EnterpriseID, s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.CanceledAt, s.EndedAt, s.StripeCustomerID, s.StripeSubscriptionID,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *subscriptionRepository) UpdateSubscription(ctx context.Context, s *domain.EnterpriseSubscription) error {
	query := `
		UPDATE veritas_enterprise_subscriptions SET
			plan_id = $1, status = $2, current_period_start = $3, current_period_end = $4,
			cancel_at_period_end = $5, canceled_at = $6, ended_at = $7,
			stripe_customer_id = $8, stripe_subscription_id = $9, updated_at = $10
		WHERE id = $11
	`
	s.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, query,
		s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.CanceledAt, s.EndedAt,
		s.StripeCustomerID, s.StripeSubscriptionID, s.UpdatedAt, s.ID,
	)
	return err
}
