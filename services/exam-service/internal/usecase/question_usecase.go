package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type questionUsecase struct {
	pool *pgxpool.Pool
	repo domain.QuestionRepository
}

func NewQuestionUsecase(pool *pgxpool.Pool, repo domain.QuestionRepository) domain.QuestionUsecase {
	return &questionUsecase{
		pool: pool,
		repo: repo,
	}
}

func (uc *questionUsecase) CreateQuestion(ctx context.Context, q *sdomain.Question, userID uuid.UUID) (*sdomain.Question, error) {
	if q.Title == "" || q.Content == "" {
		return nil, domain.ErrInvalidQuestion
	}

	q.CreatedBy = userID
	q.IsActive = true

	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		return uc.repo.WithTx(tx).Create(ctx, q)
	})
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (uc *questionUsecase) GetQuestions(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Question], error) {
	return uc.repo.ListByEnterprise(ctx, enterpriseID, params)
}

func (uc *questionUsecase) GetQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Question, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *questionUsecase) UpdateQuestion(ctx context.Context, q *sdomain.Question, userID uuid.UUID) error {
	if q.Title == "" || q.Content == "" {
		return domain.ErrInvalidQuestion
	}

	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// Fetch existing to ensure it exists
		_, err := uc.repo.WithTx(tx).GetByID(ctx, q.ID, q.EnterpriseID)
		if err != nil {
			return err
		}

		return uc.repo.WithTx(tx).Update(ctx, q)
	})
}

func (uc *questionUsecase) DeleteQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return uc.repo.Delete(ctx, id, enterpriseID)
}
