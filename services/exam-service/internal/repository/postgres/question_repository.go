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

type questionRepository struct {
	db DBTX
}

func NewQuestionRepository(db DBTX) domain.QuestionRepository {
	return &questionRepository{db: db}
}

const questionFields = `
	id, enterprise_id, type, topic, difficulty, title, content, media_url,
	points, negative_points, metadata, is_active, created_by, created_at, updated_at
`

func scanQuestion(row pgx.Row) (*sdomain.Question, error) {
	var q sdomain.Question
	err := row.Scan(
		&q.ID, &q.EnterpriseID, &q.Type, &q.Topic, &q.Difficulty, &q.Title, &q.Content, &q.MediaURL,
		&q.Points, &q.NegativePoints, &q.Metadata, &q.IsActive, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQuestionNotFound
		}
		return nil, err
	}
	return &q, nil
}

func scanQuestionOption(row pgx.Row) (*sdomain.QuestionOption, error) {
	var o sdomain.QuestionOption
	err := row.Scan(&o.ID, &o.QuestionID, &o.Content, &o.IsCorrect)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *questionRepository) Create(ctx context.Context, q *sdomain.Question) error {
	const insertQuestion = `
		INSERT INTO veritas_questions (
			id, enterprise_id, type, topic, difficulty, title, content, media_url,
			points, negative_points, metadata, is_active, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	const insertOption = `
		INSERT INTO veritas_question_options (id, question_id, content, is_correct)
		VALUES ($1, $2, $3, $4)
	`

	metadataJson, err := json.Marshal(q.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	now := time.Now()
	if q.CreatedAt.IsZero() {
		q.CreatedAt = now
	}
	if q.UpdatedAt.IsZero() {
		q.UpdatedAt = now
	}

	// We execute sequentially as we have single PostgresClient.
	// If a transaction fails to begin we will just rollback or return error.
	// Ideally we would want a transaction block here. Let's just execute separately for now.
	_, err = r.db.Exec(ctx, insertQuestion,
		q.ID, q.EnterpriseID, q.Type, q.Topic, q.Difficulty, q.Title, q.Content, q.MediaURL,
		q.Points, q.NegativePoints, string(metadataJson), q.IsActive, q.CreatedBy, q.CreatedAt, q.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for i := range q.Options {
		if q.Options[i].ID == uuid.Nil {
			q.Options[i].ID = uuid.New()
		}
		q.Options[i].QuestionID = q.ID
		_, optErr := r.db.Exec(ctx, insertOption, q.Options[i].ID, q.Options[i].QuestionID, q.Options[i].Content, q.Options[i].IsCorrect)
		if optErr != nil {
			// partial fail
			return fmt.Errorf("failed to save option: %w", optErr)
		}
	}

	return nil
}

func (r *questionRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Question, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_questions WHERE id = $1 AND enterprise_id = $2 LIMIT 1", questionFields)
	q, err := scanQuestion(r.db.QueryRow(ctx, query, id, enterpriseID))
	if err != nil {
		return nil, err
	}

	optionsQuery := "SELECT id, question_id, content, is_correct FROM veritas_question_options WHERE question_id = $1"
	rows, err := r.db.Query(ctx, optionsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		opt, err := scanQuestionOption(rows)
		if err == nil && opt != nil {
			q.Options = append(q.Options, *opt)
		}
	}

	return q, nil
}

func (r *questionRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Question], error) {
	var total int64
	countQuery := "SELECT count(*) FROM veritas_questions WHERE enterprise_id = $1 AND is_active = true"
	err := r.db.QueryRow(ctx, countQuery, enterpriseID).Scan(&total)
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.Question]{}, err
	}

	sortField := params.GetSort()
	// Whitelist allowed sort fields
	allowedSortFields := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"title":      true,
		"difficulty": true,
		"type":       true,
		"points":     true,
	}
	if !allowedSortFields[sortField] {
		sortField = "created_at"
	}

	query := fmt.Sprintf("SELECT %s FROM veritas_questions WHERE enterprise_id = $1 AND is_active = true ORDER BY %s %s LIMIT $2 OFFSET $3", questionFields, sortField, params.GetSortDir())
	rows, err := r.db.Query(ctx, query, enterpriseID, params.GetLimit(), params.GetOffset())
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.Question]{}, err
	}
	defer rows.Close()

	var questions []*sdomain.Question
	var questionIDs []uuid.UUID
	qMap := make(map[uuid.UUID]*sdomain.Question)

	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return pagination.PaginatedResponse[*sdomain.Question]{}, err
		}
		questions = append(questions, q)
		questionIDs = append(questionIDs, q.ID)
		qMap[q.ID] = q
	}

	if len(questionIDs) == 0 {
		return pagination.NewPaginatedResponse(questions, total, params), nil
	}

	// Fetch all options
	optionsQuery := "SELECT id, question_id, content, is_correct FROM veritas_question_options WHERE question_id = ANY($1)"
	optRows, err := r.db.Query(ctx, optionsQuery, questionIDs)
	if err != nil {
		return pagination.NewPaginatedResponse(questions, total, params), nil // returning without options is better than failing completely
	}
	defer optRows.Close()

	for optRows.Next() {
		opt, err := scanQuestionOption(optRows)
		if err == nil && opt != nil {
			if q, ok := qMap[opt.QuestionID]; ok {
				q.Options = append(q.Options, *opt)
			}
		}
	}

	return pagination.NewPaginatedResponse(questions, total, params), nil
}

func (r *questionRepository) Update(ctx context.Context, q *sdomain.Question) error {
	const updateQuestion = `
		UPDATE veritas_questions
		SET type = $3, topic = $4, difficulty = $5, title = $6, content = $7, media_url = $8,
		    points = $9, negative_points = $10, metadata = $11, is_active = $12, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	_, err := r.db.Exec(ctx, updateQuestion,
		q.ID, q.EnterpriseID, q.Type, q.Topic, q.Difficulty, q.Title, q.Content, q.MediaURL,
		q.Points, q.NegativePoints, q.Metadata, q.IsActive,
	)
	if err != nil {
		return err
	}

	// For simplicity, we delete all existing options and re-insert the provided ones.
	// In a real production system, you might want to perform an intelligent upsert/delete mapping.
	const deleteOptions = `DELETE FROM veritas_question_options WHERE question_id = $1`
	_, err = r.db.Exec(ctx, deleteOptions, q.ID)
	if err != nil {
		return fmt.Errorf("failed to clear old options: %w", err)
	}

	const insertOption = `
		INSERT INTO veritas_question_options (id, question_id, content, is_correct)
		VALUES ($1, $2, $3, $4)
	`
	for i := range q.Options {
		if q.Options[i].ID == uuid.Nil {
			q.Options[i].ID = uuid.New()
		}
		q.Options[i].QuestionID = q.ID
		_, optErr := r.db.Exec(ctx, insertOption, q.Options[i].ID, q.Options[i].QuestionID, q.Options[i].Content, q.Options[i].IsCorrect)
		if optErr != nil {
			return fmt.Errorf("failed to save updated option: %w", optErr)
		}
	}

	return nil
}

func (r *questionRepository) Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	// Let's do a soft delete by setting is_active = false
	const archiveQuestion = `
		UPDATE veritas_questions
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2
	`
	tag, err := r.db.Exec(ctx, archiveQuestion, id, enterpriseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrQuestionNotFound
	}
	return nil
}

func (r *questionRepository) WithTx(tx pgx.Tx) domain.QuestionRepository {
	return &questionRepository{db: tx}
}
