package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
)

type candidateRepository struct {
	db postgres.PostgresClient
}

func NewCandidateRepository(db postgres.PostgresClient) domain.CandidateRepository {
	return &candidateRepository{db: db}
}

const candidateFields = `
	id, enterprise_id, external_id, first_name, last_name, email,
	face_reference_url, is_active, created_at
`

func scanCandidate(row postgres.Row) (*domain.CandidateProfile, error) {
	var c domain.CandidateProfile
	err := row.Scan(
		&c.ID, &c.EnterpriseID, &c.ExternalID, &c.FirstName, &c.LastName, &c.Email,
		&c.FaceReferenceURL, &c.IsActive, &c.CreatedAt,
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
			face_reference_url, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, insertQuery,
		c.ID, c.EnterpriseID, c.ExternalID, c.FirstName, c.LastName, c.Email,
		c.FaceReferenceURL, c.IsActive, c.CreatedAt,
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
	// For bulk insert, pgx CopyFrom is usually best.
	// But sticking to shared db interface constraints, we'll iterate or construct a batch if supported.
	// Assumes postgres.PostgresClient provides a way or we just iterate within a transaction if possible.
	// For now, iterate with Exec.

	// Ideally we'd use pgx.Batch or QueryBuilder here.
	for _, c := range candidates {
		err := r.Create(ctx, c)
		if err != nil && !errors.Is(err, domain.ErrDuplicateExternalID) {
			return fmt.Errorf("bulk insert failed on external_id %s: %w", c.ExternalID, err)
		}
		// If duplicate, it means we can skip or update. Let's assume ignore for now on bulk insert conflicts
		// by using ON CONFLICT DO NOTHING natively, but since Create doesn't, we'd ignore the error.
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

func (r *candidateRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.CandidateProfile, error) {
	query := fmt.Sprintf("SELECT %s FROM candidate_profiles WHERE enterprise_id = $1 ORDER BY created_at DESC", candidateFields)
	rows, err := r.db.Query(ctx, query, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.CandidateProfile
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *candidateRepository) Update(ctx context.Context, c *domain.CandidateProfile) error {
	const updateQuery = `
		UPDATE candidate_profiles
		SET first_name = $3, last_name = $4, email = $5, face_reference_url = $6, is_active = $7
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery,
		c.ID, c.EnterpriseID, c.FirstName, c.LastName, c.Email, c.FaceReferenceURL, c.IsActive,
	)
	if err != nil {
		return err
	}
	if tag == 0 {
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
	if tag == 0 {
		return domain.ErrCandidateNotFound
	}
	return nil
}
