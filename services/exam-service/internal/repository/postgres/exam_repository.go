package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type examRepository struct {
	db DBTX
}

func NewExamRepository(db DBTX) domain.ExamRepository {
	return &examRepository{db: db}
}

const examFields = `
	id, enterprise_id, title, description, duration_minutes, passing_score_percent,
	negative_marking, max_participants, invitation_method, status, template_source_id,
	scheduled_start, scheduled_end, settings, created_by, created_at, updated_at
`

func scanExam(row pgx.Row) (*domain.Exam, error) {
	var e domain.Exam
	err := row.Scan(
		&e.ID, &e.EnterpriseID, &e.Title, &e.Description, &e.DurationMinutes, &e.PassingScorePercent,
		&e.NegativeMarking, &e.MaxParticipants, &e.InvitationMethod, &e.Status, &e.TemplateSourceID,
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

func scanExamQuestion(row pgx.Row) (*domain.ExamQuestion, error) {
	var eq domain.ExamQuestion
	err := row.Scan(&eq.ID, &eq.ExamID, &eq.QuestionID, &eq.PointsOverride, &eq.OrderIndex)
	if err != nil {
		return nil, err
	}
	return &eq, nil
}

func scanExamRandomizationRule(row pgx.Row) (*domain.ExamRandomizationRule, error) {
	var r domain.ExamRandomizationRule
	err := row.Scan(&r.ID, &r.ExamID, &r.Topic, &r.Difficulty, &r.QuestionCount)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *examRepository) Create(ctx context.Context, e *domain.Exam) error {
	const insertExam = `
		INSERT INTO veritas_exams (
			id, enterprise_id, title, description, duration_minutes, passing_score_percent,
			negative_marking, max_participants, invitation_method, status, template_source_id,
			scheduled_start, scheduled_end, settings, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
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

	_, err := r.db.Exec(ctx, insertExam,
		e.ID, e.EnterpriseID, e.Title, e.Description, e.DurationMinutes, e.PassingScorePercent,
		e.NegativeMarking, e.MaxParticipants, e.InvitationMethod, e.Status, e.TemplateSourceID,
		e.ScheduledStart, e.ScheduledEnd, e.Settings, e.CreatedBy, e.CreatedAt, e.UpdatedAt,
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

func (r *examRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.Exam, error) {
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

func (r *examRepository) Update(ctx context.Context, e *domain.Exam) error {
	const updateExam = `
		UPDATE veritas_exams
		SET title = $3, description = $4, duration_minutes = $5, passing_score_percent = $6,
		    negative_marking = $7, max_participants = $8, invitation_method = $9, status = $10,
		    scheduled_start = $11, scheduled_end = $12, settings = $13, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	_, err := r.db.Exec(ctx, updateExam,
		e.ID, e.EnterpriseID, e.Title, e.Description, e.DurationMinutes, e.PassingScorePercent,
		e.NegativeMarking, e.MaxParticipants, e.InvitationMethod, e.Status,
		e.ScheduledStart, e.ScheduledEnd, e.Settings,
	)
	return err
}

func (r *examRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Exam, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_exams WHERE enterprise_id = $1 ORDER BY created_at DESC", examFields)
	rows, err := r.db.Query(ctx, query, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exams []*domain.Exam
	for rows.Next() {
		e, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		// Notice: To keep this lightweight, we aren't joining questions/rules on a bulk list request.
		// If the client needs deep details, they should call GetByID.
		exams = append(exams, e)
	}

	return exams, nil
}

func (r *examRepository) Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	// Let's do a soft delete via status 'Archived' (domain.ExamArchived)
	const archiveExam = `
		UPDATE veritas_exams
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, archiveExam, id, enterpriseID, domain.ExamArchived)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrExamNotFound
	}
	return nil
}

func (r *examRepository) AddQuestion(ctx context.Context, examID uuid.UUID, eq *domain.ExamQuestion) error {
	if eq.ID == uuid.Nil {
		eq.ID = uuid.New()
	}
	eq.ExamID = examID

	const insertEq = `
		INSERT INTO veritas_exam_questions (id, exam_id, question_id, points_override, order_index)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, insertEq, eq.ID, eq.ExamID, eq.QuestionID, eq.PointsOverride, eq.OrderIndex)
	return err
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

func (r *examRepository) UpdateQuestionMapping(ctx context.Context, examID uuid.UUID, eq *domain.ExamQuestion) error {
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

func (r *examRepository) AddRandomizationRule(ctx context.Context, examID uuid.UUID, rule *domain.ExamRandomizationRule) error {
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

func (r *examRepository) UpdateRandomizationRule(ctx context.Context, examID uuid.UUID, rule *domain.ExamRandomizationRule) error {
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
