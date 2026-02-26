package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type questionUsecase struct {
	repo domain.QuestionRepository
}

func NewQuestionUsecase(repo domain.QuestionRepository) domain.QuestionUsecase {
	return &questionUsecase{repo: repo}
}

func (uc *questionUsecase) CreateQuestion(ctx context.Context, q *domain.Question, userID uuid.UUID) (*domain.Question, error) {
	if q.Title == "" || q.Content == "" {
		return nil, domain.ErrInvalidQuestion
	}

	q.CreatedBy = userID
	q.IsActive = true

	if err := uc.repo.Create(ctx, q); err != nil {
		return nil, err
	}

	return q, nil
}

func (uc *questionUsecase) GetQuestions(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Question, error) {
	return uc.repo.ListByEnterprise(ctx, enterpriseID)
}

func (uc *questionUsecase) GetQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.Question, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *questionUsecase) UpdateQuestion(ctx context.Context, q *domain.Question, userID uuid.UUID) error {
	if q.Title == "" || q.Content == "" {
		return domain.ErrInvalidQuestion
	}

	// Fetch existing to ensure it exists and we aren't altering metadata that shouldn't be altered
	// optionally
	_, err := uc.repo.GetByID(ctx, q.ID, q.EnterpriseID)
	if err != nil {
		return err
	}

	return uc.repo.Update(ctx, q)
}

func (uc *questionUsecase) DeleteQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return uc.repo.Delete(ctx, id, enterpriseID)
}
