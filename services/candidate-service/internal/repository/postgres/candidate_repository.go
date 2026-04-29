package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// allowedCandidateSortFields restricts which columns can be used in ORDER BY
// to prevent SQL injection from query params.
var allowedCandidateSortFields = map[string]string{
	"created_at":  "created_at",
	"first_name":  "first_name",
	"last_name":   "last_name",
	"external_id": "external_id",
}

func safeCandidateSortField(s string) string {
	if col, ok := allowedCandidateSortFields[s]; ok {
		return col
	}
	return "created_at"
}

type candidateRepository struct {
	db DBTX
}

func NewCandidateRepository(db DBTX) domain.CandidateRepository {
	return &candidateRepository{db: db}
}

const candidateFields = `
	id, enterprise_id, external_id, first_name, last_name, email,
	is_active, created_at
`

func scanCandidate(row pgx.Row) (*domain.CandidateProfile, error) {
	var c domain.CandidateProfile
	err := row.Scan(
		&c.ID, &c.EnterpriseID, &c.ExternalID, &c.FirstName, &c.LastName, &c.Email,
		&c.IsActive, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCandidateNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *candidateRepository) Create(ctx context.Context, c *domain.CandidateProfile) error {
	const insertQuery = `
		INSERT INTO candidate_profiles (
			id, enterprise_id, external_id, first_name, last_name, email,
			is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, insertQuery,
		c.ID, c.EnterpriseID, c.ExternalID, c.FirstName, c.LastName, c.Email,
		c.IsActive, c.CreatedAt,
	)
	if err != nil {
		// PostgreSQL duplicate key violation
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"uq_candidate_external\" (SQLSTATE 23505)" {
			return domain.ErrDuplicateExternalID
		}
		return err
	}
	return nil
}

func (r *candidateRepository) CreateBulk(ctx context.Context, candidates []*domain.CandidateProfile) error {
	if len(candidates) == 0 {
		return nil
	}

	cols := []string{"id", "enterprise_id", "external_id", "first_name", "last_name", "email", "is_active", "created_at"}
	rows := make([][]any, 0, len(candidates))

	for _, c := range candidates {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		if c.CreatedAt.IsZero() {
			c.CreatedAt = time.Now()
		}
		rows = append(rows, []any{
			c.ID, c.EnterpriseID, c.ExternalID, c.FirstName, c.LastName, c.Email,
			c.IsActive, c.CreatedAt,
		})
	}

	conn, ok := r.db.(interface {
		CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	})
	if !ok {
		return domain.ErrNotSupported
	}

	_, err := conn.CopyFrom(
		ctx,
		pgx.Identifier{"candidate_profiles"},
		cols,
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"uq_candidate_external\" (SQLSTATE 23505)" {
			return domain.ErrDuplicateExternalID
		}
		return fmt.Errorf("bulk upload: %w", err)
	}

	return nil
}

func (r *candidateRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.CandidateProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM candidate_profiles WHERE id = $1 AND enterprise_id = $2 LIMIT 1", candidateFields)
	return scanCandidate(r.db.QueryRow(ctx, query, id, enterpriseID))
}

func (r *candidateRepository) GetByExternalID(ctx context.Context, externalID string, enterpriseID uuid.UUID) (*domain.CandidateProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM candidate_profiles WHERE external_id = $1 AND enterprise_id = $2 LIMIT 1", candidateFields)
	return scanCandidate(r.db.QueryRow(ctx, query, externalID, enterpriseID))
}

func (r *candidateRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.CandidateProfile, int64, error) {
	// Count query
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM candidate_profiles WHERE enterprise_id = $1", enterpriseID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	sortCol := safeCandidateSortField(params.GetSort())
	query := fmt.Sprintf(
		"SELECT %s FROM candidate_profiles WHERE enterprise_id = $1 ORDER BY %s %s LIMIT $2 OFFSET $3",
		candidateFields, sortCol, params.GetSortDir(),
	)
	rows, err := r.db.Query(ctx, query, enterpriseID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.CandidateProfile
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, nil
}

func (r *candidateRepository) Update(ctx context.Context, c *domain.CandidateProfile) error {
	const updateQuery = `
		UPDATE candidate_profiles
		SET first_name = $3, last_name = $4, email = $5
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery,
		c.ID, c.EnterpriseID, c.FirstName, c.LastName, c.Email,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCandidateNotFound
	}
	return nil
}

func (r *candidateRepository) Deactivate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	const updateQuery = `
		UPDATE candidate_profiles
		SET is_active = false
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery, id, enterpriseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCandidateNotFound
	}
	return nil
}

func (r *candidateRepository) WithTx(tx pgx.Tx) domain.CandidateRepository {
	return &candidateRepository{db: tx}
}

func (r *candidateRepository) GetEmailsByExamID(ctx context.Context, examID, enterpriseID uuid.UUID) ([]string, error) {
	const query = `
		SELECT cp.email
		FROM candidate_profiles cp
		JOIN exam_enrollments ee ON cp.id = ee.candidate_id
		WHERE ee.exam_id = $1 AND ee.enterprise_id = $2 AND cp.email IS NOT NULL
	`
	rows, err := r.db.Query(ctx, query, examID, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, nil
}
