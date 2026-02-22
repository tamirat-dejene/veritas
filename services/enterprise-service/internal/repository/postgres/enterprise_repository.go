package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
)

type enterpriseRepository struct {
	db postgres.PostgresClient
}

func NewEnterpriseRepository(db postgres.PostgresClient) domain.EnterpriseRepository {
	return &enterpriseRepository{db: db}
}

const enterpriseFields = `
	id, slug, display_name, legal_name, contact_email, owner_account_id, status, approved_at,
	suspended_at, deleted_at, subscription_plan_id, subscription_status, current_period_start,
	current_period_end, logo_url, primary_color, secondary_color, custom_domain, contact_phone,
	address_line1, address_line2, city, country, settings, created_at, updated_at, created_by, updated_by
`

func scanEnterprise(row postgres.Row) (*domain.Enterprise, error) {
	var e domain.Enterprise
	err := row.Scan(
		&e.ID, &e.Slug, &e.DisplayName, &e.LegalName, &e.ContactEmail, &e.OwnerAccountID, &e.Status, &e.ApprovedAt,
		&e.SuspendedAt, &e.DeletedAt, &e.SubscriptionPlanID, &e.SubscriptionStatus, &e.CurrentPeriodStart,
		&e.CurrentPeriodEnd, &e.LogoURL, &e.PrimaryColor, &e.SecondaryColor, &e.CustomDomain, &e.ContactPhone,
		&e.AddressLine1, &e.AddressLine2, &e.City, &e.Country, &e.Settings, &e.CreatedAt, &e.UpdatedAt,
		&e.CreatedBy, &e.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEnterpriseNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *enterpriseRepository) Create(ctx context.Context, e *domain.Enterprise) error {
	const query = `
		INSERT INTO veritas_enterprise (
			id, slug, display_name, legal_name, contact_email, owner_account_id, status,
			logo_url, primary_color, secondary_color, custom_domain, contact_phone,
			address_line1, address_line2, city, country, settings,
			created_at, updated_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}

	_, err := r.db.Exec(ctx, query,
		e.ID, e.Slug, e.DisplayName, e.LegalName, e.ContactEmail, e.OwnerAccountID, e.Status,
		e.LogoURL, e.PrimaryColor, e.SecondaryColor, e.CustomDomain, e.ContactPhone,
		e.AddressLine1, e.AddressLine2, e.City, e.Country, e.Settings,
		e.CreatedAt, e.UpdatedAt, e.CreatedBy, e.UpdatedBy,
	)
	return err
}

func (r *enterpriseRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Enterprise, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_enterprise WHERE id = $1 LIMIT 1", enterpriseFields)
	row := r.db.QueryRow(ctx, query, id)
	return scanEnterprise(row)
}

func (r *enterpriseRepository) FindBySlug(ctx context.Context, slug string) (*domain.Enterprise, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_enterprise WHERE slug = $1 LIMIT 1", enterpriseFields)
	row := r.db.QueryRow(ctx, query, slug)
	return scanEnterprise(row)
}

func (r *enterpriseRepository) Update(ctx context.Context, e *domain.Enterprise) error {
	const query = `
		UPDATE veritas_enterprise
		SET slug = $2, display_name = $3, legal_name = $4, contact_email = $5, status = $6,
		    approved_at = $7, suspended_at = $8, deleted_at = $9, subscription_plan_id = $10,
		    subscription_status = $11, current_period_start = $12, current_period_end = $13,
		    logo_url = $14, primary_color = $15, secondary_color = $16, custom_domain = $17,
		    contact_phone = $18, address_line1 = $19, address_line2 = $20, city = $21,
		    country = $22, settings = $23, updated_at = NOW(), updated_by = $24
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		e.ID, e.Slug, e.DisplayName, e.LegalName, e.ContactEmail, e.Status,
		e.ApprovedAt, e.SuspendedAt, e.DeletedAt, e.SubscriptionPlanID,
		e.SubscriptionStatus, e.CurrentPeriodStart, e.CurrentPeriodEnd,
		e.LogoURL, e.PrimaryColor, e.SecondaryColor, e.CustomDomain,
		e.ContactPhone, e.AddressLine1, e.AddressLine2, e.City,
		e.Country, e.Settings, e.UpdatedBy,
	)
	return err
}

func (r *enterpriseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE veritas_enterprise SET status = 'Deleted', deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *enterpriseRepository) List(ctx context.Context, filter map[string]any) ([]*domain.Enterprise, error) {
	// Simple list for now, can be expanded with filters
	query := fmt.Sprintf("SELECT %s FROM veritas_enterprise", enterpriseFields)
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enterprises []*domain.Enterprise
	for rows.Next() {
		e, err := scanEnterprise(rows)
		if err != nil {
			return nil, err
		}
		enterprises = append(enterprises, e)
	}
	return enterprises, rows.Err()
}
