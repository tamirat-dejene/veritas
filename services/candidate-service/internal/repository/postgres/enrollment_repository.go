package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
)

type enrollmentRepository struct {
	db DBTX
}

func NewEnrollmentRepository(db DBTX) domain.EnrollmentRepository {
	return &enrollmentRepository{db: db}
}

const enrollmentFields = `
	id, enterprise_id, exam_id, candidate_id, invitation_method, access_token_hash,
	token_expires_at, max_attempts, attempts_used, status, created_at
`

func scanEnrollment(row pgx.Row) (*domain.ExamEnrollment, error) {
	var e domain.ExamEnrollment
	err := row.Scan(
		&e.ID, &e.EnterpriseID, &e.ExamID, &e.CandidateID, &e.InvitationMethod,
		&e.AccessTokenHash, &e.TokenExpiresAt, &e.MaxAttempts, &e.AttemptsUsed,
		&e.Status, &e.CreatedAt,
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
			id, enterprise_id, exam_id, candidate_id, invitation_method, access_token_hash,
			token_expires_at, max_attempts, attempts_used, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, insertQuery,
		e.ID, e.EnterpriseID, e.ExamID, e.CandidateID, e.InvitationMethod,
		e.AccessTokenHash, e.TokenExpiresAt, e.MaxAttempts, e.AttemptsUsed,
		e.Status, e.CreatedAt,
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

func (r *enrollmentRepository) ListByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) ([]*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE exam_id = $1 AND enterprise_id = $2 ORDER BY created_at DESC", enrollmentFields)
	rows, err := r.db.Query(ctx, query, examID, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ExamEnrollment
	for rows.Next() {
		e, err := scanEnrollment(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

func (r *enrollmentRepository) Update(ctx context.Context, e *domain.ExamEnrollment) error {
	const updateQuery = `
		UPDATE exam_enrollments
		SET access_token_hash = $3, token_expires_at = $4, max_attempts = $5,
		    status = $6
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery,
		e.ID, e.EnterpriseID, e.AccessTokenHash, e.TokenExpiresAt, e.MaxAttempts, e.Status,
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
		SET attempts_used = attempts_used + 1, status = 'Attempted'
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
