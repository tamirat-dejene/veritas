package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

const (
	planFields = "id, name, slug, description, price, currency, billing_cycle, features, is_active, created_at, updated_at"
	subFields  = "id, enterprise_id, plan_id, status, current_period_start, current_period_end, cancel_at_period_end, canceled_at, ended_at, stripe_customer_id, stripe_subscription_id, created_at, updated_at"
)

type subscriptionRepository struct {
	db DBTX
}

func NewSubscriptionRepository(db DBTX) domain.SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) ListPlans(ctx context.Context, params pagination.Params) ([]*domain.SubscriptionPlan, int64, error) {
	var total int64
	countQuery := "SELECT COUNT(*) FROM veritas_subscription_plans WHERE is_active = true"
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortCol := "created_at"
	switch params.GetSort() {
	case "price", "name":
		sortCol = params.GetSort()
	}

	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans WHERE is_active = true ORDER BY %s %s LIMIT $1 OFFSET $2", planFields, sortCol, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, params.GetLimit(), params.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []*domain.SubscriptionPlan
	for rows.Next() {
		var p domain.SubscriptionPlan
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		plans = append(plans, &p)
	}
	return plans, total, nil
}

func (r *subscriptionRepository) ListAllPlans(ctx context.Context, params pagination.Params) ([]*domain.SubscriptionPlan, int64, error) {
	var total int64
	countQuery := "SELECT COUNT(*) FROM veritas_subscription_plans"
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	sortCol := "created_at"
	switch params.GetSort() {
	case "price", "name":
		sortCol = params.GetSort()
	}

	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans ORDER BY %s %s LIMIT $1 OFFSET $2", planFields, sortCol, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, params.GetLimit(), params.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []*domain.SubscriptionPlan
	for rows.Next() {
		var p domain.SubscriptionPlan
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		plans = append(plans, &p)
	}
	return plans, total, nil
}

func (r *subscriptionRepository) GetPlanByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_subscription_plans WHERE id = $1", planFields)
	var p domain.SubscriptionPlan
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.Currency, &p.BillingCycle, &p.Features, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
		if errors.Is(err, pgx.ErrNoRows) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *subscriptionRepository) GetSubscriptionByStripeID(ctx context.Context, stripeSubscriptionID string) (*domain.EnterpriseSubscription, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_enterprise_subscriptions WHERE stripe_subscription_id = $1", subFields)
	var s domain.EnterpriseSubscription
	err := r.db.QueryRow(ctx, query, stripeSubscriptionID).Scan(
		&s.ID, &s.EnterpriseID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd, &s.CanceledAt, &s.EndedAt, &s.StripeCustomerID, &s.StripeSubscriptionID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrSubscriptionAlreadyExists
		}
		return err
	}
	return nil
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
	cmd, err := r.db.Exec(ctx, query,
		s.PlanID, s.Status, s.CurrentPeriodStart, s.CurrentPeriodEnd,
		s.CancelAtPeriodEnd, s.CanceledAt, s.EndedAt,
		s.StripeCustomerID, s.StripeSubscriptionID, s.UpdatedAt, s.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrSubscriptionNotFound
	}
	return nil
}

func (r *subscriptionRepository) CreatePlan(ctx context.Context, p *domain.SubscriptionPlan) error {
	query := `
		INSERT INTO veritas_subscription_plans (
			id, name, slug, description, price, currency, billing_cycle, features, stripe_price_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		p.ID, p.Name, p.Slug, p.Description, p.Price, p.Currency, p.BillingCycle, p.Features, p.StripePriceID, p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrPlanAlreadyExists
		}
		return err
	}
	return nil
}

func (r *subscriptionRepository) UpdatePlan(ctx context.Context, p *domain.SubscriptionPlan) error {
	query := `
		UPDATE veritas_subscription_plans SET
			name = $1, slug = $2, description = $3, price = $4, currency = $5,
			billing_cycle = $6, features = $7, stripe_price_id = $8, is_active = $9, updated_at = $10
		WHERE id = $11
	`
	p.UpdatedAt = time.Now()
	cmd, err := r.db.Exec(ctx, query,
		p.Name, p.Slug, p.Description, p.Price, p.Currency, p.BillingCycle, p.Features, p.StripePriceID, p.IsActive, p.UpdatedAt, p.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrPlanAlreadyExists
		}
		return err
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrPlanNotFound
	}
	return nil
}

func (r *subscriptionRepository) GetLapsedSubscriptions(ctx context.Context, limit int) ([]*domain.EnterpriseSubscription, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM veritas_enterprise_subscriptions
		WHERE status IN ('Active', 'Trial')
		  AND current_period_end <= NOW()
		  AND cancel_at_period_end = true
		LIMIT $1
	`, subFields)
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.EnterpriseSubscription
	for rows.Next() {
		var s domain.EnterpriseSubscription
		if err := rows.Scan(
			&s.ID, &s.EnterpriseID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CancelAtPeriodEnd, &s.CanceledAt, &s.EndedAt, &s.StripeCustomerID, &s.StripeSubscriptionID,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *subscriptionRepository) GetPastDueCandidates(ctx context.Context, limit int) ([]*domain.EnterpriseSubscription, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM veritas_enterprise_subscriptions
		WHERE status IN ('Active', 'Trial')
		  AND current_period_end <= NOW()
		  AND cancel_at_period_end = false
		LIMIT $1
	`, subFields)
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*domain.EnterpriseSubscription
	for rows.Next() {
		var s domain.EnterpriseSubscription
		if err := rows.Scan(
			&s.ID, &s.EnterpriseID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
			&s.CancelAtPeriodEnd, &s.CanceledAt, &s.EndedAt, &s.StripeCustomerID, &s.StripeSubscriptionID,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, &s)
	}
	return subs, nil
}

func (r *subscriptionRepository) WithTx(tx pgx.Tx) domain.SubscriptionRepository {
	return &subscriptionRepository{db: tx}
}
