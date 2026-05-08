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

var allowedSessionSortFields = map[string]string{
	"created_at": "created_at",
	"status":     "status",
	"started_at": "started_at",
}

func safeSessionSortField(s string) string {
	if col, ok := allowedSessionSortFields[s]; ok {
		return col
	}
	return "created_at"
}

var allowedSubmissionSortFields = map[string]string{
	"created_at":   "created_at",
	"submitted_at": "submitted_at",
}

func safeSubmissionSortField(s string) string {
	if col, ok := allowedSubmissionSortFields[s]; ok {
		return col
	}
	return "created_at"
}

type sessionRepository struct {
	db DBTX
}

func NewSessionRepository(db DBTX) domain.SessionRepository {
	return &sessionRepository{db: db}
}

// ---------------------------------------------------------
// Sessions
// ---------------------------------------------------------

const sessionFields = `
	id, enterprise_id, exam_id, candidate_id, enrollment_id, status,
	started_at, expires_at, submitted_at, terminated_at, termination_reason,
	client_ip::text, user_agent, face_registered_url, created_at
`

func scanSession(row pgx.Row) (*domain.ExamSession, error) {
	var s domain.ExamSession
	err := row.Scan(
		&s.ID, &s.EnterpriseID, &s.ExamID, &s.CandidateID, &s.EnrollmentID, &s.Status,
		&s.StartedAt, &s.ExpiresAt, &s.SubmittedAt, &s.TerminatedAt, &s.TerminationReason,
		&s.ClientIP, &s.UserAgent, &s.FaceRegisteredURL, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) CreateSession(ctx context.Context, s *domain.ExamSession) error {
	const insertQuery = `
		INSERT INTO exam_sessions (
			id, enterprise_id, exam_id, candidate_id, enrollment_id, status,
			started_at, expires_at, client_ip, user_agent, face_registered_url, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = s.CreatedAt
	}

	_, err := r.db.Exec(ctx, insertQuery,
		s.ID, s.EnterpriseID, s.ExamID, s.CandidateID, s.EnrollmentID, s.Status,
		s.StartedAt, s.ExpiresAt, s.ClientIP, s.UserAgent, s.FaceRegisteredURL, s.CreatedAt,
	)
	return err
}

func (r *sessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamSession, error) {
	where := "id = $1"
	args := []interface{}{id}
	if enterpriseID != uuid.Nil {
		where += " AND enterprise_id = $2"
		args = append(args, enterpriseID)
	}
	query := fmt.Sprintf("SELECT %s FROM exam_sessions WHERE %s LIMIT 1", sessionFields, where)
	return scanSession(r.db.QueryRow(ctx, query, args...))
}

func (r *sessionRepository) GetSessionByEnrollment(ctx context.Context, enrollmentID uuid.UUID) (*domain.ExamSession, error) {
	// enrollmentID is already globally unique across enterprises in our initial schema assumptions, but let's keep it consistent
	query := fmt.Sprintf("SELECT %s FROM exam_sessions WHERE enrollment_id = $1 LIMIT 1", sessionFields)
	return scanSession(r.db.QueryRow(ctx, query, enrollmentID))
}

func (r *sessionRepository) ListSessionsByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, status *domain.SessionStatus, params pagination.Params) ([]*domain.ExamSession, int64, error) {
	// Build WHERE clause (reused for count and data)
	where := "exam_id = $1 AND enterprise_id = $2"
	countArgs := []interface{}{examID, enterpriseID}
	dataArgs := []interface{}{examID, enterpriseID}

	if status != nil {
		where += " AND status = $3"
		countArgs = append(countArgs, *status)
		dataArgs = append(dataArgs, *status)
	}

	// Count query
	var total int64
	if err := r.db.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM exam_sessions WHERE %s", where),
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query — append LIMIT and OFFSET as the next positional args
	next := len(dataArgs) + 1
	sortCol := safeSessionSortField(params.GetSort())
	dataArgs = append(dataArgs, params.GetLimit(), params.GetOffset())
	query := fmt.Sprintf(
		"SELECT %s FROM exam_sessions WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		sessionFields, where, sortCol, params.GetSortDir(), next, next+1,
	)
	rows, err := r.db.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.ExamSession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, s)
	}
	return list, total, nil
}

func (r *sessionRepository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus, reason *string) error {
	const updateQuery = `
		UPDATE exam_sessions
		SET status = $2::session_status, termination_reason = COALESCE($3, termination_reason), 
		    terminated_at = CASE WHEN $2::session_status = 'Terminated' THEN NOW() ELSE terminated_at END,
		    submitted_at = CASE WHEN $2::session_status = 'Submitted' THEN NOW() ELSE submitted_at END
		WHERE id = $1
	`
	tag, err := r.db.Exec(ctx, updateQuery, id, status, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}
	return nil
}

func (r *sessionRepository) TerminateActiveSessionsByEnterprise(ctx context.Context, enterpriseID uuid.UUID, reason string) error {
	const updateQuery = `
		UPDATE exam_sessions
		SET status = 'Terminated'::session_status, 
		    termination_reason = $2, 
		    terminated_at = NOW()
		WHERE enterprise_id = $1 AND status = 'Active'::session_status
	`
	_, err := r.db.Exec(ctx, updateQuery, enterpriseID, reason)
	return err
}

func (r *sessionRepository) TerminateActiveSessionsByExam(ctx context.Context, examID uuid.UUID, reason string) error {
	const updateQuery = `
		UPDATE exam_sessions
		SET status = 'Terminated'::session_status, 
		    termination_reason = $2, 
		    terminated_at = NOW()
		WHERE exam_id = $1 AND status = 'Active'::session_status
	`
	_, err := r.db.Exec(ctx, updateQuery, examID, reason)
	return err
}

// ---------------------------------------------------------
// Session Questions (Snapshots)
// ---------------------------------------------------------

func (r *sessionRepository) SaveQuestionsSnapshot(ctx context.Context, sessionID uuid.UUID, questions []domain.SessionQuestion) error {
	// Simple iterative insert. In production with huge exams, consider QueryBuilder/pgx.Batch
	const insertQuery = `
		INSERT INTO session_questions (id, session_id, question_id, question_snapshot, order_index, points, negative_points)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for _, q := range questions {
		if q.ID == uuid.Nil {
			q.ID = uuid.New()
		}
		q.SessionID = sessionID
		_, err := r.db.Exec(ctx, insertQuery, q.ID, q.SessionID, q.QuestionID, q.QuestionSnapshot, q.OrderIndex, q.Points, q.NegativePoints)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *sessionRepository) GetSessionQuestions(ctx context.Context, sessionID uuid.UUID) ([]domain.SessionQuestion, error) {
	query := `
		SELECT id, session_id, question_id, question_snapshot, order_index, points, negative_points
		FROM session_questions WHERE session_id = $1 ORDER BY order_index ASC
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SessionQuestion
	for rows.Next() {
		var q domain.SessionQuestion
		if err := rows.Scan(&q.ID, &q.SessionID, &q.QuestionID, &q.QuestionSnapshot, &q.OrderIndex, &q.Points, &q.NegativePoints); err != nil {
			return nil, err
		}
		list = append(list, q)
	}
	return list, nil
}

func (r *sessionRepository) GetSessionQuestion(ctx context.Context, sessionID uuid.UUID, sessionQuestionID uuid.UUID) (*domain.SessionQuestion, error) {
	query := `
		SELECT id, session_id, question_id, question_snapshot, order_index, points, negative_points
		FROM session_questions WHERE session_id = $1 AND id = $2 LIMIT 1
	`
	var q domain.SessionQuestion
	err := r.db.QueryRow(ctx, query, sessionID, sessionQuestionID).Scan(
		&q.ID, &q.SessionID, &q.QuestionID, &q.QuestionSnapshot, &q.OrderIndex, &q.Points, &q.NegativePoints,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQuestionNotFound
		}
		return nil, err
	}
	return &q, nil
}

// ---------------------------------------------------------
// Answers
// ---------------------------------------------------------

func (r *sessionRepository) UpsertAnswer(ctx context.Context, a *domain.SessionAnswer) error {
	const upsertQuery = `
		INSERT INTO session_answers (id, session_id, session_question_id, answer_data, is_final, saved_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT ON CONSTRAINT uq_session_question DO UPDATE 
		SET answer_data = EXCLUDED.answer_data,
		    is_final = EXCLUDED.is_final,
		    saved_at = EXCLUDED.saved_at
	`
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.SavedAt = time.Now()

	_, err := r.db.Exec(ctx, upsertQuery, a.ID, a.SessionID, a.SessionQuestionID, a.AnswerData, a.IsFinal, a.SavedAt)
	return err
}

func (r *sessionRepository) GetSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]domain.SessionAnswer, error) {
	query := `
		SELECT id, session_id, session_question_id, answer_data, is_final, saved_at
		FROM session_answers WHERE session_id = $1
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.SessionAnswer
	for rows.Next() {
		var a domain.SessionAnswer
		if err := rows.Scan(&a.ID, &a.SessionID, &a.SessionQuestionID, &a.AnswerData, &a.IsFinal, &a.SavedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

// ---------------------------------------------------------
// Submissions
// ---------------------------------------------------------

func (r *sessionRepository) CreateSubmission(ctx context.Context, sub *domain.ExamSubmission) error {
	const insertQuery = `
		INSERT INTO exam_submissions (id, session_id, submitted_at, auto_submitted, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, insertQuery, sub.ID, sub.SessionID, sub.SubmittedAt, sub.AutoSubmitted, sub.CreatedAt)
	if err != nil {
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"exam_submissions_session_id_key\" (SQLSTATE 23505)" {
			return domain.ErrSubmissionExists
		}
		return err
	}
	return nil
}

func (r *sessionRepository) GetSubmissionBySession(ctx context.Context, sessionID uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamSubmission, error) {
	where := "es.session_id = $1"
	args := []interface{}{sessionID}
	if enterpriseID != uuid.Nil {
		where += " AND s.enterprise_id = $2"
		args = append(args, enterpriseID)
	}

	query := fmt.Sprintf(`
		SELECT es.id, es.session_id, es.submitted_at, es.auto_submitted, es.created_at
		FROM exam_submissions es
		JOIN exam_sessions s ON es.session_id = s.id
		WHERE %s LIMIT 1
	`, where)

	var s domain.ExamSubmission
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.SessionID, &s.SubmittedAt, &s.AutoSubmitted, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) GetSubmissionByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamSubmission, error) {
	where := "es.id = $1"
	args := []interface{}{id}
	if enterpriseID != uuid.Nil {
		where += " AND s.enterprise_id = $2"
		args = append(args, enterpriseID)
	}

	query := fmt.Sprintf(`
		SELECT es.id, es.session_id, es.submitted_at, es.auto_submitted, es.created_at
		FROM exam_submissions es
		JOIN exam_sessions s ON es.session_id = s.id
		WHERE %s LIMIT 1
	`, where)

	var s domain.ExamSubmission
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&s.ID, &s.SessionID, &s.SubmittedAt, &s.AutoSubmitted, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) GetSubmissionsByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamSubmission, int64, error) {
	const baseJoin = `
		FROM exam_submissions es
		JOIN exam_sessions s ON es.session_id = s.id
		WHERE s.exam_id = $1 AND s.enterprise_id = $2
	`

	// Count query
	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) "+baseJoin, examID, enterpriseID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query with pagination
	sortCol := safeSubmissionSortField(params.GetSort())
	query := fmt.Sprintf(
		`SELECT es.id, es.session_id, es.submitted_at, es.auto_submitted, es.created_at %s ORDER BY es.%s %s LIMIT $3 OFFSET $4`,
		baseJoin, sortCol, params.GetSortDir(),
	)
	rows, err := r.db.Query(ctx, query, examID, enterpriseID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*domain.ExamSubmission
	for rows.Next() {
		var s domain.ExamSubmission
		if err := rows.Scan(&s.ID, &s.SessionID, &s.SubmittedAt, &s.AutoSubmitted, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, &s)
	}
	return list, total, nil
}

func (r *sessionRepository) CountSessionsByEnterpriseAndStatus(ctx context.Context, enterpriseID uuid.UUID, status domain.SessionStatus) (int, error) {
	var count int
	query := "SELECT count(*) FROM exam_sessions WHERE enterprise_id = $1 AND status = $2"
	err := r.db.QueryRow(ctx, query, enterpriseID, status).Scan(&count)
	return count, err
}

func (r *sessionRepository) GetExpiredActiveSessions(ctx context.Context, limit int) ([]*domain.ExamSession, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM exam_sessions
		WHERE status = 'Active' AND expires_at <= NOW()
		ORDER BY expires_at ASC
		LIMIT $1
	`, sessionFields)

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ExamSession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *sessionRepository) WithTx(tx pgx.Tx) domain.SessionRepository {
	return &sessionRepository{db: tx}
}
