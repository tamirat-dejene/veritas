package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type examRepository struct {
	db DBTX
}

func NewExamRepository(db DBTX) domain.ExamRepository {
	return &examRepository{db: db}
}

const examFields = `
	id, enterprise_id, title, description, duration_minutes, passing_score_percent,
	negative_marking, max_participants, status, template_source_id,
	scheduled_start, scheduled_end, settings, created_by, created_at, updated_at
`

func scanExam(row pgx.Row) (*sdomain.Exam, error) {
	var e sdomain.Exam
	err := row.Scan(
		&e.ID, &e.EnterpriseID, &e.Title, &e.Description, &e.DurationMinutes, &e.PassingScorePercent,
		&e.NegativeMarking, &e.MaxParticipants, &e.Status, &e.TemplateSourceID,
		&e.ScheduledStart, &e.ScheduledEnd, &e.Settings, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExamNotFound
		}
		return nil, err
	}
	return &e, nil
}

func scanExamQuestion(row pgx.Row) (*sdomain.ExamQuestion, error) {
	var eq sdomain.ExamQuestion
	err := row.Scan(&eq.ID, &eq.ExamID, &eq.QuestionID, &eq.OrderIndex)
	if err != nil {
		return nil, err
	}
	return &eq, nil
}


func (r *examRepository) Create(ctx context.Context, e *sdomain.Exam) error {
	const insertExam = `
		INSERT INTO veritas_exams (
			id, enterprise_id, title, description, duration_minutes, passing_score_percent,
			negative_marking, max_participants, status, template_source_id,
			scheduled_start, scheduled_end, settings, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	settingsJson, err := json.Marshal(e.Settings)
	if err != nil {
		return fmt.Errorf("%w: exam settings: %v", domain.ErrMarshalFailed, err)
	}
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

	_, err = r.db.Exec(ctx, insertExam,
		e.ID, e.EnterpriseID, e.Title, e.Description, e.DurationMinutes, e.PassingScorePercent,
		e.NegativeMarking, e.MaxParticipants, e.Status, e.TemplateSourceID,
		e.ScheduledStart, e.ScheduledEnd, settingsJson, e.CreatedBy, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Insert ExamQuestions
	for i := range e.Questions {
		if e.Questions[i].ID == uuid.Nil {
			e.Questions[i].ID = uuid.New()
		}
		e.Questions[i].ExamID = e.ID
		_, optErr := r.db.Exec(ctx, `INSERT INTO veritas_exam_questions (id, exam_id, question_id, order_index) VALUES ($1, $2, $3, $4)`,
			e.Questions[i].ID, e.Questions[i].ExamID, e.Questions[i].QuestionID, e.Questions[i].OrderIndex)
		if optErr != nil {
			return fmt.Errorf("%w: exam question: %v", domain.ErrInternal, optErr)
		}
	}


	return nil
}

func (r *examRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_exams WHERE id = $1 AND enterprise_id = $2 LIMIT 1", examFields)
	e, err := scanExam(r.db.QueryRow(ctx, query, id, enterpriseID))
	if err != nil {
		return nil, err
	}

	// Get associated Questions Mapping
	qRows, err := r.db.Query(ctx, "SELECT id, exam_id, question_id, order_index FROM veritas_exam_questions WHERE exam_id = $1", id)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			eq, eqErr := scanExamQuestion(qRows)
			if eqErr == nil && eq != nil {
				e.Questions = append(e.Questions, *eq)
			}
		}
	}


	return e, nil
}

func (r *examRepository) Update(ctx context.Context, e *sdomain.Exam) error {
	const updateExam = `
		UPDATE veritas_exams
		SET title = $3, description = $4, duration_minutes = $5, passing_score_percent = $6,
		    negative_marking = $7, max_participants = $8, status = $9,
		    scheduled_start = $10, scheduled_end = $11, settings = $12, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	settingsJson, err := json.Marshal(e.Settings)
	if err != nil {
		return fmt.Errorf("%w: exam settings: %v", domain.ErrMarshalFailed, err)
	}

	_, err = r.db.Exec(ctx, updateExam,
		e.ID, e.EnterpriseID, e.Title, e.Description, e.DurationMinutes, e.PassingScorePercent,
		e.NegativeMarking, e.MaxParticipants, e.Status,
		e.ScheduledStart, e.ScheduledEnd, settingsJson,
	)
	return err
}

func (r *examRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params, search string, archived bool, archivedOnly bool) (pagination.PaginatedResponse[*sdomain.Exam], error) {
	args := []any{enterpriseID}

	// Apply archived option
	whereClause := "enterprise_id = $1"
	if archivedOnly {
		whereClause += " AND status = 'Archived'"
	} else if !archived {
		whereClause += " AND status != 'Archived'"
	}

	// Optional title search — append only when a non-empty search term is provided.
	if search != "" {
		args = append(args, "%"+search+"%")
		whereClause += fmt.Sprintf(" AND title ILIKE $%d", len(args))
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT count(*) FROM veritas_exams WHERE %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.PaginatedResponse[*sdomain.Exam]{}, err
	}

	sortField := params.GetSort()
	allowedSortFields := map[string]bool{
		"created_at":            true,
		"updated_at":            true,
		"title":                 true,
		"duration_minutes":      true,
		"passing_score_percent": true,
		"status":                true,
	}
	if !allowedSortFields[sortField] {
		sortField = "created_at"
	}

	// Append LIMIT / OFFSET args after the WHERE args.
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, params.GetLimit(), params.GetOffset())

	query := fmt.Sprintf(
		"SELECT %s FROM veritas_exams WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		examFields, whereClause, sortField, params.GetSortDir(), limitIdx, offsetIdx,
	)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.Exam]{}, err
	}
	defer rows.Close()

	var exams []*sdomain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return pagination.PaginatedResponse[*sdomain.Exam]{}, err
		}
		exams = append(exams, e)
	}

	return pagination.NewPaginatedResponse(exams, total, params), nil
}

func (r *examRepository) Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	// Let's do a soft delete via status 'Archived' (sdomain.ExamArchived)
	const archiveExam = `
		UPDATE veritas_exams
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, archiveExam, id, enterpriseID, sdomain.ExamArchived)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrExamNotFound
	}
	return nil
}

func (r *examRepository) CountByEnterpriseAndStatus(ctx context.Context, enterpriseID uuid.UUID, status sdomain.ExamStatus) (int, error) {
	var count int
	query := "SELECT count(*) FROM veritas_exams WHERE enterprise_id = $1 AND status = $2"
	err := r.db.QueryRow(ctx, query, enterpriseID, status).Scan(&count)
	return count, err
}


func (r *examRepository) AddQuestions(ctx context.Context, examID uuid.UUID, eqs []*sdomain.ExamQuestion) error {
	const insertEq = `
		INSERT INTO veritas_exam_questions (id, exam_id, question_id, order_index)
		VALUES ($1, $2, $3, $4)
	`
	for _, eq := range eqs {
		if eq.ID == uuid.Nil {
			eq.ID = uuid.New()
		}
		eq.ExamID = examID
		_, err := r.db.Exec(ctx, insertEq, eq.ID, eq.ExamID, eq.QuestionID, eq.OrderIndex)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *examRepository) GetExamQuestions(ctx context.Context, examID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.ExamQuestion], error) {
	var total int64
	err := r.db.QueryRow(ctx, "SELECT count(*) FROM veritas_exam_questions WHERE exam_id = $1", examID).Scan(&total)
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
	}

	sortField := params.GetSort()
	// Map allowed columns for sorting safely
	sortField = "order_index"

	query := fmt.Sprintf("SELECT id, exam_id, question_id, order_index FROM veritas_exam_questions WHERE exam_id = $1 ORDER BY %s %s NULLS LAST LIMIT $2 OFFSET $3", sortField, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, examID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
	}
	defer rows.Close()

	var eqs []*sdomain.ExamQuestion
	for rows.Next() {
		var eq sdomain.ExamQuestion
		if err := rows.Scan(&eq.ID, &eq.ExamID, &eq.QuestionID, &eq.OrderIndex); err != nil {
			return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
		}
		eqs = append(eqs, &eq)
	}

	return pagination.NewPaginatedResponse(eqs, total, params), nil
}

func (r *examRepository) RemoveQuestion(ctx context.Context, examID uuid.UUID, questionID uuid.UUID) error {
	const deleteEq = `DELETE FROM veritas_exam_questions WHERE exam_id = $1 AND question_id = $2`
	tag, err := r.db.Exec(ctx, deleteEq, examID, questionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMappingNotFound
	}
	return nil
}

func (r *examRepository) UpdateQuestionMapping(ctx context.Context, examID uuid.UUID, eq *sdomain.ExamQuestion) error {
	const updateEq = `
		UPDATE veritas_exam_questions
		SET order_index = $3
		WHERE exam_id = $1 AND question_id = $2
	`
	tag, err := r.db.Exec(ctx, updateEq, examID, eq.QuestionID, eq.OrderIndex)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMappingNotFound
	}
	return nil
}


func (r *examRepository) GetScheduledExamsDue(ctx context.Context, limit int) ([]*sdomain.Exam, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM veritas_exams
		WHERE status = 'Scheduled' AND scheduled_start <= NOW()
		LIMIT $1
	`, examFields)
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exams []*sdomain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, e)
	}
	return exams, nil
}

func (r *examRepository) GetActiveExamsPastEnd(ctx context.Context, limit int) ([]*sdomain.Exam, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM veritas_exams
		WHERE status = 'Active' AND scheduled_end <= NOW()
		LIMIT $1
	`, examFields)
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exams []*sdomain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, e)
	}
	return exams, nil
}

func (r *examRepository) GetStaleClosedExams(ctx context.Context, cutoff time.Time, limit int) ([]*sdomain.Exam, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM veritas_exams
		WHERE status = 'Closed' AND updated_at < $1
		LIMIT $2
	`, examFields)
	rows, err := r.db.Query(ctx, query, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exams []*sdomain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, e)
	}
	return exams, nil
}

// GetByEnterpriseAndStatuses returns all exams for an enterprise whose status
// is contained in the provided statuses slice. This is intentionally unbounded
// (no LIMIT) so that consistency operations process every affected exam.
func (r *examRepository) GetByEnterpriseAndStatuses(ctx context.Context, enterpriseID uuid.UUID, statuses []sdomain.ExamStatus) ([]*sdomain.Exam, error) {
	if len(statuses) == 0 {
		return nil, nil
	}

	// Convert []sdomain.ExamStatus to []string for pgx ANY() binding.
	statusStrings := make([]string, len(statuses))
	for i, s := range statuses {
		statusStrings[i] = string(s)
	}

	query := fmt.Sprintf(`
		SELECT %s FROM veritas_exams
		WHERE enterprise_id = $1 AND status = ANY($2)
		ORDER BY created_at ASC
	`, examFields)

	rows, err := r.db.Query(ctx, query, enterpriseID, statusStrings)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exams []*sdomain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, e)
	}
	return exams, nil
}

func (r *examRepository) WithTx(tx pgx.Tx) domain.ExamRepository {
	return &examRepository{db: tx}
}

