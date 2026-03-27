package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type QuestionRepository interface {
	Create(ctx context.Context, q *sdomain.Question) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, withCorrectAnswer bool) (*sdomain.Question, error)
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params, withCorrectAnswer bool) (pagination.PaginatedResponse[*sdomain.Question], error)
	Update(ctx context.Context, q *sdomain.Question) error
	Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	WithTx(tx pgx.Tx) QuestionRepository
}

type QuestionUsecase interface {
	CreateQuestion(ctx context.Context, q *sdomain.Question, userID uuid.UUID) (*sdomain.Question, error)
	GetQuestions(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params, withCorrectAnswer bool) (pagination.PaginatedResponse[*sdomain.Question], error)
	GetQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, withCorrectAnswer bool) (*sdomain.Question, error)
	UpdateQuestion(ctx context.Context, q *sdomain.Question, userID uuid.UUID) error
	DeleteQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}
