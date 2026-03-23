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

var allowedEnrollmentSortFields = map[string]string{
	"created_at":   "created_at",
	"attempts_used": "attempts_used",
}

func safeEnrollmentSortField(s string) string {
	if col, ok := allowedEnrollmentSortFields[s]; ok {
		return col
	}
	return "created_at"
}

type enrollmentRepository struct {
	db DBTX
}

func NewEnrollmentRepository(db DBTX) domain.EnrollmentRepository {
	return &enrollmentRepository{db: db}
}

const enrollmentFields = `
	id, enterprise_id, exam_id, candidate_id, access_token_hash,
	token_expires_at, max_attempts, attempts_used, created_at
`

func scanEnrollment(row pgx.Row) (*domain.ExamEnrollment, error) {
	var e domain.ExamEnrollment
	err := row.Scan(
		&e.ID, &e.EnterpriseID, &e.ExamID, &e.CandidateID,
		&e.AccessTokenHash, &e.TokenExpiresAt, &e.MaxAttempts, &e.AttemptsUsed,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEnrollmentNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *enrollmentRepository) Create(ctx context.Context, e *domain.ExamEnrollment) error {
	const insertQuery = `
		INSERT INTO exam_enrollments (
			id, enterprise_id, exam_id, candidate_id, access_token_hash,
			token_expires_at, max_attempts, attempts_used, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, insertQuery,
		e.ID, e.EnterpriseID, e.ExamID, e.CandidateID,
		e.AccessTokenHash, e.TokenExpiresAt, e.MaxAttempts, e.AttemptsUsed,
		e.CreatedAt,
	)
	// Optionally check duplicate enrollment violation
	return err
}

func (r *enrollmentRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE id = $1 AND enterprise_id = $2 LIMIT 1", enrollmentFields)
	return scanEnrollment(r.db.QueryRow(ctx, query, id, enterpriseID))
}

func (r *enrollmentRepository) GetByExamAndCandidate(ctx context.Context, examID uuid.UUID, candidateID uuid.UUID) (*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE exam_id = $1 AND candidate_id = $2 LIMIT 1", enrollmentFields)
	return scanEnrollment(r.db.QueryRow(ctx, query, examID, candidateID))
}

func (r *enrollmentRepository) ListByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamEnrollment, int64, error) {
	// Count query
	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM exam_enrollments WHERE exam_id = $1 AND enterprise_id = $2",
		examID, enterpriseID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	sortCol := safeEnrollmentSortField(params.GetSort())
	query := fmt.Sprintf(
		"SELECT %s FROM exam_enrollments WHERE exam_id = $1 AND enterprise_id = $2 ORDER BY %s %s LIMIT $3 OFFSET $4",
		enrollmentFields, sortCol, params.GetSortDir(),
	)
	rows, err := r.db.Query(ctx, query, examID, enterpriseID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.ExamEnrollment
	for rows.Next() {
		e, err := scanEnrollment(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, e)
	}
	return list, total, nil
}

func (r *enrollmentRepository) Update(ctx context.Context, e *domain.ExamEnrollment) error {
	const updateQuery = `
		UPDATE exam_enrollments
		SET access_token_hash = $3, token_expires_at = $4, max_attempts = $5
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery,
		e.ID, e.EnterpriseID, e.AccessTokenHash, e.TokenExpiresAt, e.MaxAttempts,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	const updateQuery = `
		UPDATE exam_enrollments
		SET attempts_used = attempts_used + 1
		WHERE id = $1
	`
	tag, err := r.db.Exec(ctx, updateQuery, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) WithTx(tx pgx.Tx) domain.EnrollmentRepository {
	return &enrollmentRepository{db: tx}
}
