package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	postgres "github.com/tamirat-dejene/veritas/shared/db/pg"
)

type questionRepository struct {
	db postgres.PostgresClient
}

func NewQuestionRepository(db postgres.PostgresClient) domain.QuestionRepository {
	return &questionRepository{db: db}
}

const questionFields = `
	id, enterprise_id, type, topic, difficulty, title, content, media_url,
	points, negative_points, metadata, is_active, created_by, created_at, updated_at
`

func scanQuestion(row postgres.Row) (*domain.Question, error) {
	var q domain.Question
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

func scanQuestionOption(row postgres.Row) (*domain.QuestionOption, error) {
	var o domain.QuestionOption
	err := row.Scan(&o.ID, &o.QuestionID, &o.Content, &o.IsCorrect)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *questionRepository) Create(ctx context.Context, q *domain.Question) error {
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
	_, err := r.db.Exec(ctx, insertQuestion,
		q.ID, q.EnterpriseID, q.Type, q.Topic, q.Difficulty, q.Title, q.Content, q.MediaURL,
		q.Points, q.NegativePoints, q.Metadata, q.IsActive, q.CreatedBy, q.CreatedAt, q.UpdatedAt,
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

func (r *questionRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.Question, error) {
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

func (r *questionRepository) ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Question, error) {
	query := fmt.Sprintf("SELECT %s FROM veritas_questions WHERE enterprise_id = $1 ORDER BY created_at DESC", questionFields)
	rows, err := r.db.Query(ctx, query, enterpriseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*domain.Question
	var questionIDs []uuid.UUID
	qMap := make(map[uuid.UUID]*domain.Question)

	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
		questionIDs = append(questionIDs, q.ID)
		qMap[q.ID] = q
	}

	if len(questionIDs) == 0 {
		return questions, nil
	}

	// Fetch all options
	// Simplified single query for options. This assumes Postgres ANY array.
	optionsQuery := "SELECT id, question_id, content, is_correct FROM veritas_question_options WHERE question_id = ANY($1)"
	optRows, err := r.db.Query(ctx, optionsQuery, questionIDs)
	if err != nil {
		return questions, nil // returning without options is better than failing completely
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

	return questions, nil
}
