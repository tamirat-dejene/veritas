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
	err := row.Scan(&eq.ID, &eq.ExamID, &eq.QuestionID, &eq.PointsOverride, &eq.OrderIndex)
	if err != nil {
		return nil, err
	}
	return &eq, nil
}

func scanExamRandomizationRule(row pgx.Row) (*sdomain.ExamRandomizationRule, error) {
	var r sdomain.ExamRandomizationRule
	err := row.Scan(&r.ID, &r.ExamID, &r.Topic, &r.Difficulty, &r.QuestionCount)
	if err != nil {
		return nil, err
	}
	return &r, nil
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
		return fmt.Errorf("failed to marshal exam settings: %w", err)
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
		_, optErr := r.db.Exec(ctx, `INSERT INTO veritas_exam_questions (id, exam_id, question_id, points_override, order_index) VALUES ($1, $2, $3, $4, $5)`,
			e.Questions[i].ID, e.Questions[i].ExamID, e.Questions[i].QuestionID, e.Questions[i].PointsOverride, e.Questions[i].OrderIndex)
		if optErr != nil {
			return fmt.Errorf("failed to save exam question: %w", optErr)
		}
	}

	// Insert RandomizationRules
	for i := range e.RandomizationRules {
		if e.RandomizationRules[i].ID == uuid.Nil {
			e.RandomizationRules[i].ID = uuid.New()
		}
		e.RandomizationRules[i].ExamID = e.ID
		_, rErr := r.db.Exec(ctx, `INSERT INTO veritas_exam_randomization_rules (id, exam_id, topic, difficulty, question_count) VALUES ($1, $2, $3, $4, $5)`,
			e.RandomizationRules[i].ID, e.RandomizationRules[i].ExamID, e.RandomizationRules[i].Topic, e.RandomizationRules[i].Difficulty, e.RandomizationRules[i].QuestionCount)
		if rErr != nil {
			return fmt.Errorf("failed to save randomization rule: %w", rErr)
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
	qRows, err := r.db.Query(ctx, "SELECT id, exam_id, question_id, points_override, order_index FROM veritas_exam_questions WHERE exam_id = $1", id)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			eq, eqErr := scanExamQuestion(qRows)
			if eqErr == nil && eq != nil {
				e.Questions = append(e.Questions, *eq)
			}
		}
	}

	// Get associated Randomization rules
	rRows, err := r.db.Query(ctx, "SELECT id, exam_id, topic, difficulty, question_count FROM veritas_exam_randomization_rules WHERE exam_id = $1", id)
	if err == nil {
		defer rRows.Close()
		for rRows.Next() {
			rr, rrErr := scanExamRandomizationRule(rRows)
			if rrErr == nil && rr != nil {
				e.RandomizationRules = append(e.RandomizationRules, *rr)
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
		return fmt.Errorf("failed to marshal exam settings: %w", err)
	}

	_, err = r.db.Exec(ctx, updateExam,
		e.ID, e.EnterpriseID, e.Title, e.Description, e.DurationMinutes, e.PassingScorePercent,
		e.NegativeMarking, e.MaxParticipants, e.Status,
		e.ScheduledStart, e.ScheduledEnd, settingsJson,
	)
	return err
}

func (r *examRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Exam], error) {
	var total int64
	countQuery := "SELECT count(*) FROM veritas_exams WHERE enterprise_id = $1 AND status != 'Archived'"
	err := r.db.QueryRow(ctx, countQuery, enterpriseID).Scan(&total)
	if err != nil {
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

	query := fmt.Sprintf("SELECT %s FROM veritas_exams WHERE enterprise_id = $1 AND status != 'Archived' ORDER BY %s %s LIMIT $2 OFFSET $3", examFields, sortField, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, enterpriseID, params.GetLimit(), params.GetOffset())
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
		// Notice: To keep this lightweight, we aren't joining questions/rules on a bulk list request.
		// If the client needs deep details, they should call GetByID.
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

func (r *examRepository) AddQuestions(ctx context.Context, examID uuid.UUID, eqs []*sdomain.ExamQuestion) error {
	const insertEq = `
		INSERT INTO veritas_exam_questions (id, exam_id, question_id, points_override, order_index)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, eq := range eqs {
		if eq.ID == uuid.Nil {
			eq.ID = uuid.New()
		}
		eq.ExamID = examID
		_, err := r.db.Exec(ctx, insertEq, eq.ID, eq.ExamID, eq.QuestionID, eq.PointsOverride, eq.OrderIndex)
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
	if sortField != "points_override" {
		sortField = "order_index"
	}

	query := fmt.Sprintf("SELECT id, exam_id, question_id, points_override, order_index FROM veritas_exam_questions WHERE exam_id = $1 ORDER BY %s %s NULLS LAST LIMIT $2 OFFSET $3", sortField, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, examID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
	}
	defer rows.Close()

	var eqs []*sdomain.ExamQuestion
	for rows.Next() {
		var eq sdomain.ExamQuestion
		if err := rows.Scan(&eq.ID, &eq.ExamID, &eq.QuestionID, &eq.PointsOverride, &eq.OrderIndex); err != nil {
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
		return fmt.Errorf("exam question mapping not found")
	}
	return nil
}

func (r *examRepository) UpdateQuestionMapping(ctx context.Context, examID uuid.UUID, eq *sdomain.ExamQuestion) error {
	const updateEq = `
		UPDATE veritas_exam_questions
		SET points_override = $3, order_index = $4
		WHERE exam_id = $1 AND question_id = $2
	`
	tag, err := r.db.Exec(ctx, updateEq, examID, eq.QuestionID, eq.PointsOverride, eq.OrderIndex)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("exam question mapping not found")
	}
	return nil
}

func (r *examRepository) AddRandomizationRule(ctx context.Context, examID uuid.UUID, rule *sdomain.ExamRandomizationRule) error {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.ExamID = examID

	const insertRule = `
		INSERT INTO veritas_exam_randomization_rules (id, exam_id, topic, difficulty, question_count)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, insertRule, rule.ID, rule.ExamID, rule.Topic, rule.Difficulty, rule.QuestionCount)
	return err
}

func (r *examRepository) UpdateRandomizationRule(ctx context.Context, examID uuid.UUID, rule *sdomain.ExamRandomizationRule) error {
	const updateRule = `
		UPDATE veritas_exam_randomization_rules
		SET topic = $3, difficulty = $4, question_count = $5
		WHERE exam_id = $1 AND id = $2
	`
	tag, err := r.db.Exec(ctx, updateRule, examID, rule.ID, rule.Topic, rule.Difficulty, rule.QuestionCount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("randomization rule not found")
	}
	return nil
}

func (r *examRepository) DeleteRandomizationRule(ctx context.Context, examID uuid.UUID, ruleID uuid.UUID) error {
	const deleteRule = `DELETE FROM veritas_exam_randomization_rules WHERE exam_id = $1 AND id = $2`
	tag, err := r.db.Exec(ctx, deleteRule, examID, ruleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("randomization rule not found")
	}
	return nil
}

func (r *examRepository) WithTx(tx pgx.Tx) domain.ExamRepository {
	return &examRepository{db: tx}
}
