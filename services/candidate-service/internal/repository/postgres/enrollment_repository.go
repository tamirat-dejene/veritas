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
	"created_at":    "created_at",
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

// enrollmentFields is the canonical SELECT column list — order must match scanEnrollment.
const enrollmentFields = `
	id, enterprise_id, exam_id, candidate_id,
	access_token_hash, invitation_code_hash,
	token_expires_at, max_attempts, attempts_used,
	status, invitation_sent_at, created_at
`

func scanEnrollment(row pgx.Row) (*domain.ExamEnrollment, error) {
	var e domain.ExamEnrollment
	err := row.Scan(
		&e.ID, &e.EnterpriseID, &e.ExamID, &e.CandidateID,
		&e.AccessTokenHash, &e.InvitationCodeHash,
		&e.TokenExpiresAt, &e.MaxAttempts, &e.AttemptsUsed,
		&e.Status, &e.InvitationSentAt, &e.CreatedAt,
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
			id, enterprise_id, exam_id, candidate_id,
			access_token_hash, invitation_code_hash,
			token_expires_at, max_attempts, attempts_used,
			status, invitation_sent_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if e.Status == "" {
		e.Status = domain.StatusPending
	}

	_, err := r.db.Exec(ctx, insertQuery,
		e.ID, e.EnterpriseID, e.ExamID, e.CandidateID,
		e.AccessTokenHash, e.InvitationCodeHash,
		e.TokenExpiresAt, e.MaxAttempts, e.AttemptsUsed,
		e.Status, e.InvitationSentAt, e.CreatedAt,
	)
	return err
}

func (r *enrollmentRepository) CreateBulk(ctx context.Context, enrollments []*domain.ExamEnrollment) error {
	if len(enrollments) == 0 {
		return nil
	}

	cols := []string{
		"id", "enterprise_id", "exam_id", "candidate_id",
		"access_token_hash", "invitation_code_hash",
		"token_expires_at", "max_attempts", "attempts_used",
		"status", "invitation_sent_at", "created_at",
	}
	rows := make([][]any, 0, len(enrollments))

	for _, e := range enrollments {
		if e.ID == uuid.Nil {
			e.ID = uuid.New()
		}
		if e.CreatedAt.IsZero() {
			e.CreatedAt = time.Now()
		}
		if e.Status == "" {
			e.Status = domain.StatusPending
		}
		rows = append(rows, []any{
			e.ID, e.EnterpriseID, e.ExamID, e.CandidateID,
			e.AccessTokenHash, e.InvitationCodeHash,
			e.TokenExpiresAt, e.MaxAttempts, e.AttemptsUsed,
			e.Status, e.InvitationSentAt, e.CreatedAt,
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
		pgx.Identifier{"exam_enrollments"},
		cols,
		pgx.CopyFromRows(rows),
	)
	return err
}

func (r *enrollmentRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE id = $1 AND enterprise_id = $2 LIMIT 1", enrollmentFields)
	return scanEnrollment(r.db.QueryRow(ctx, query, id, enterpriseID))
}

func (r *enrollmentRepository) GetByIDs(ctx context.Context, ids []uuid.UUID, enterpriseID uuid.UUID) ([]*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE id = ANY($1) AND enterprise_id = $2", enrollmentFields)
	rows, err := r.db.Query(ctx, query, ids, enterpriseID)
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

func (r *enrollmentRepository) GetByExamAndCandidate(ctx context.Context, examID uuid.UUID, candidateID uuid.UUID) (*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE exam_id = $1 AND candidate_id = $2 LIMIT 1", enrollmentFields)
	return scanEnrollment(r.db.QueryRow(ctx, query, examID, candidateID))
}

func (r *enrollmentRepository) GetByExamAndCandidates(ctx context.Context, examID uuid.UUID, candidateIDs []uuid.UUID) ([]*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE exam_id = $1 AND candidate_id = ANY($2)", enrollmentFields)
	rows, err := r.db.Query(ctx, query, examID, candidateIDs)
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

func (r *enrollmentRepository) GetByInvitationCodeHash(ctx context.Context, codeHash string) (*domain.ExamEnrollment, error) {
	query := fmt.Sprintf("SELECT %s FROM exam_enrollments WHERE invitation_code_hash = $1 LIMIT 1", enrollmentFields)
	return scanEnrollment(r.db.QueryRow(ctx, query, codeHash))
}

func (r *enrollmentRepository) ListByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamEnrollment, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM exam_enrollments WHERE exam_id = $1 AND enterprise_id = $2",
		examID, enterpriseID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

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
		SET access_token_hash    = $3,
		    invitation_code_hash = $4,
		    token_expires_at     = $5,
		    max_attempts         = $6,
		    status               = $7,
		    invitation_sent_at   = $8
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, updateQuery,
		e.ID, e.EnterpriseID,
		e.AccessTokenHash, e.InvitationCodeHash,
		e.TokenExpiresAt, e.MaxAttempts,
		e.Status, e.InvitationSentAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) UpdateInvitation(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, codeHash string, invitedAt time.Time) error {
	const q = `
		UPDATE exam_enrollments 
		SET invitation_code_hash = $3, 
		    status = $4, 
		    invitation_sent_at = $5 
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, q, id, enterpriseID, codeHash, domain.StatusInvited, invitedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.EnrollmentStatus) error {
	const q = `UPDATE exam_enrollments SET status = $2 WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE exam_enrollments SET attempts_used = attempts_used + 1 WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) GetExpiredPendingEnrollments(ctx context.Context, limit int) ([]*domain.ExamEnrollment, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM exam_enrollments
		WHERE status IN ('Pending', 'Invited', 'Opened') AND token_expires_at <= NOW()
		ORDER BY token_expires_at ASC
		LIMIT $1
	`, enrollmentFields)

	rows, err := r.db.Query(ctx, query, limit)
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

func (r *enrollmentRepository) RevokeByEnterprise(ctx context.Context, enterpriseID uuid.UUID) error {
	const q = `
		UPDATE exam_enrollments 
		SET status = 'Revoked' 
		WHERE enterprise_id = $1 AND status IN ('Pending', 'Invited', 'Opened')
	`
	_, err := r.db.Exec(ctx, q, enterpriseID)
	return err
}

func (r *enrollmentRepository) RevokeByExam(ctx context.Context, examID uuid.UUID) error {
	const q = `
		UPDATE exam_enrollments 
		SET status = 'Revoked' 
		WHERE exam_id = $1 AND status IN ('Pending', 'Invited', 'Opened')
	`
	_, err := r.db.Exec(ctx, q, examID)
	return err
}

func (r *enrollmentRepository) Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	const q = "DELETE FROM exam_enrollments WHERE id = $1 AND enterprise_id = $2"
	tag, err := r.db.Exec(ctx, q, id, enterpriseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEnrollmentNotFound
	}
	return nil
}

func (r *enrollmentRepository) ResetAttempts(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE exam_enrollments SET attempts_used = 0 WHERE id = $1`
	tag, err := r.db.Exec(ctx, q, id)
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
