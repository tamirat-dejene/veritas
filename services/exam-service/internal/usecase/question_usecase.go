package usecase

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type questionUsecase struct {
	pool    *pgxpool.Pool
	repo    domain.QuestionRepository
	storage domain.FileStorage
}

func NewQuestionUsecase(pool *pgxpool.Pool, repo domain.QuestionRepository, storage domain.FileStorage) domain.QuestionUsecase {
	return &questionUsecase{
		pool:    pool,
		repo:    repo,
		storage: storage,
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

func (uc *questionUsecase) GetQuestions(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params, withCorrectAnswer bool) (pagination.PaginatedResponse[*sdomain.Question], error) {
	return uc.repo.ListByEnterprise(ctx, enterpriseID, params, withCorrectAnswer)
}

func (uc *questionUsecase) GetQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, withCorrectAnswer bool) (*sdomain.Question, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID, withCorrectAnswer)
}

func (uc *questionUsecase) UpdateQuestion(ctx context.Context, q *sdomain.Question, userID uuid.UUID) error {
	if q.Title == "" || q.Content == "" {
		return domain.ErrInvalidQuestion
	}

	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// Fetch existing to ensure it exists
		_, err := uc.repo.WithTx(tx).GetByID(ctx, q.ID, q.EnterpriseID, false)
		if err != nil {
			return err
		}

		return uc.repo.WithTx(tx).Update(ctx, q)
	})
}

func (uc *questionUsecase) DeleteQuestion(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	// 1. Fetch existing question to check for media
	q, err := uc.repo.GetByID(ctx, id, enterpriseID, false)
	if err != nil {
		return err
	}

	// 2. If it has a MediaURL, delete it from Cloudinary
	if q.MediaURL != nil && *q.MediaURL != "" {
		// Use predictable naming: q_{questionID}
		fileName := fmt.Sprintf("q_%s", id.String())
		_ = uc.storage.Delete(ctx, fileName) // non-critical failure
	}

	// 3. Perform hard delete in repository
	return uc.repo.Delete(ctx, id, enterpriseID)
}

func (uc *questionUsecase) UploadMedia(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, fileName string, content io.Reader) (string, error) {
	// 1. Verify question exists and belongs to enterprise
	_, err := uc.repo.GetByID(ctx, id, enterpriseID, false)
	if err != nil {
		return "", err
	}

	// 2. Upload to storage
	mediaURL, err := uc.storage.Upload(ctx, fileName, content)
	if err != nil {
		return "", err
	}

	// 3. Update database
	if err := uc.repo.UpdateMediaURL(ctx, id, enterpriseID, &mediaURL); err != nil {
		return "", err
	}

	return mediaURL, nil
}
