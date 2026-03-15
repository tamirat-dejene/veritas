package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type enterpriseRepository struct {
	db DBTX
}

func NewEnterpriseRepository(db DBTX) domain.EnterpriseRepository {
	return &enterpriseRepository{db: db}
}

func (r *enterpriseRepository) WithTx(tx pgx.Tx) domain.EnterpriseRepository {
	return &enterpriseRepository{db: tx}
}

const enterpriseFields = `
	id, slug, display_name, legal_name, contact_email, owner_account_id, status, approved_at,
	suspended_at, deleted_at, retention_until, subscription_plan_id, subscription_status,
	current_period_start, current_period_end, logo_url, primary_color, secondary_color,
	custom_domain, contact_phone, address_line1, address_line2, city, country, settings,
	created_at, updated_at, created_by, updated_by
`

func scanEnterprise(row pgx.Row) (*domain.Enterprise, error) {
	var e domain.Enterprise
	err := row.Scan(
		&e.ID, &e.Slug, &e.DisplayName, &e.LegalName, &e.ContactEmail, &e.OwnerAccountID, &e.Status, &e.ApprovedAt,
		&e.SuspendedAt, &e.DeletedAt, &e.RetentionUntil, &e.SubscriptionPlanID, &e.SubscriptionStatus,
		&e.CurrentPeriodStart, &e.CurrentPeriodEnd, &e.LogoURL, &e.PrimaryColor, &e.SecondaryColor,
		&e.CustomDomain, &e.ContactPhone, &e.AddressLine1, &e.AddressLine2, &e.City, &e.Country, &e.Settings,
		&e.CreatedAt, &e.UpdatedAt, &e.CreatedBy, &e.UpdatedBy,
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
		    approved_at = $7, suspended_at = $8, deleted_at = $9, retention_until = $10,
		    subscription_plan_id = $11, subscription_status = $12,
		    current_period_start = $13, current_period_end = $14,
		    logo_url = $15, primary_color = $16, secondary_color = $17, custom_domain = $18,
		    contact_phone = $19, address_line1 = $20, address_line2 = $21, city = $22,
		    country = $23, settings = $24, updated_at = NOW(), updated_by = $25
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		e.ID, e.Slug, e.DisplayName, e.LegalName, e.ContactEmail, e.Status,
		e.ApprovedAt, e.SuspendedAt, e.DeletedAt, e.RetentionUntil,
		e.SubscriptionPlanID, e.SubscriptionStatus, e.CurrentPeriodStart, e.CurrentPeriodEnd,
		e.LogoURL, e.PrimaryColor, e.SecondaryColor, e.CustomDomain,
		e.ContactPhone, e.AddressLine1, e.AddressLine2, e.City,
		e.Country, e.Settings, e.UpdatedBy,
	)
	return err
}

func (r *enterpriseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE veritas_enterprise
		SET status = 'Deleted', deleted_at = NOW(), retention_until = NOW() + INTERVAL '90 days', updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *enterpriseRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM veritas_enterprise WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// List is the legacy unfiltered list kept for backward compatibility.
func (r *enterpriseRepository) List(ctx context.Context, filter map[string]any) ([]*domain.Enterprise, error) {
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

// ListPaginated implements filtered, paginated enterprise listing.
func (r *enterpriseRepository) ListPaginated(ctx context.Context, f domain.EnterpriseFilter) ([]*domain.Enterprise, int, error) {
	var (
		whereClauses []string
		args         []any
		argIdx       = 1
	)

	if f.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *f.Status)
		argIdx++
	}
	if f.SubscriptionStatus != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("subscription_status = $%d", argIdx))
		args = append(args, *f.SubscriptionStatus)
		argIdx++
	}
	if f.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(display_name ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}

	where := ""
	if len(whereClauses) > 0 {
		where = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM veritas_enterprise %s", where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.Limit

	dataQuery := fmt.Sprintf(
		"SELECT %s FROM veritas_enterprise %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		enterpriseFields, where, argIdx, argIdx+1,
	)
	args = append(args, f.Limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enterprises []*domain.Enterprise
	for rows.Next() {
		e, err := scanEnterprise(rows)
		if err != nil {
			return nil, 0, err
		}
		enterprises = append(enterprises, e)
	}
	return enterprises, total, rows.Err()
}
