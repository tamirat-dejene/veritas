package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type QuestionType string

const (
	QuestionTypeMCQ         QuestionType = "MCQ"
	QuestionTypeTrueFalse   QuestionType = "TrueFalse"
	QuestionTypeShortAnswer QuestionType = "ShortAnswer"
	QuestionTypeEssay       QuestionType = "Essay"
)

type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "Easy"
	DifficultyMedium DifficultyLevel = "Medium"
	DifficultyHard   DifficultyLevel = "Hard"
)

type QuestionOption struct {
	ID         uuid.UUID `db:"id" json:"id"`
	QuestionID uuid.UUID `db:"question_id" json:"questionId"`
	Content    string    `db:"content" json:"content"`
	IsCorrect  bool      `db:"is_correct" json:"isCorrect"`
}

type Question struct {
	ID             uuid.UUID              `db:"id" json:"id"`
	EnterpriseID   uuid.UUID              `db:"enterprise_id" json:"enterpriseId"`
	Type           QuestionType           `db:"type" json:"type"`
	Topic          string                 `db:"topic" json:"topic"`
	Difficulty     DifficultyLevel        `db:"difficulty" json:"difficulty"`
	Title          string                 `db:"title" json:"title"`
	Content        string                 `db:"content" json:"content"`
	MediaURL       *string                `db:"media_url" json:"mediaUrl,omitempty"`
	Points         int                    `db:"points" json:"points"`
	NegativePoints float64                `db:"negative_points" json:"negativePoints"`
	Metadata       map[string]interface{} `db:"metadata" json:"metadata,omitempty"`
	IsActive       bool                   `db:"is_active" json:"isActive"`
	CreatedBy      uuid.UUID              `db:"created_by" json:"createdBy"`
	CreatedAt      time.Time              `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time              `db:"updated_at" json:"updatedAt"`

	Options []QuestionOption `json:"options,omitempty"`
}

type QuestionRepository interface {
	Create(ctx context.Context, q *Question) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*Question, error)
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Question], error)
	Update(ctx context.Context, q *Question) error
	Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	WithTx(tx pgx.Tx) QuestionRepository
}

type QuestionUsecase interface {
	CreateQuestion(ctx context.Context, q *Question, userID uuid.UUID) (*Question, error)
	GetQuestions(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*Question], error)
	GetQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*Question, error)
	UpdateQuestion(ctx context.Context, q *Question, userID uuid.UUID) error
	DeleteQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}
