package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
)

type mockExamRepository struct {
	domain.ExamRepository
	GetByIDFunc func(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error)
	UpdateFunc  func(ctx context.Context, exam *sdomain.Exam) error
}

func (m *mockExamRepository) GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error) {
	return m.GetByIDFunc(ctx, id, enterpriseID)
}

func (m *mockExamRepository) Update(ctx context.Context, exam *sdomain.Exam) error {
	return m.UpdateFunc(ctx, exam)
}

func (m *mockExamRepository) WithTx(tx pgx.Tx) domain.ExamRepository {
	return m
}

func TestValidateAndAssignOrderIndexes(t *testing.T) {
	uc := &examUsecase{}

	t.Run("empty exam, auto-assign", func(t *testing.T) {
		incoming := []*sdomain.ExamQuestion{
			{QuestionID: uuid.New()},
			{QuestionID: uuid.New()},
		}
		err := uc.validateAndAssignOrderIndexes(nil, incoming)
		assert.NoError(t, err)
		assert.Equal(t, 1, *incoming[0].OrderIndex)
		assert.Equal(t, 2, *incoming[1].OrderIndex)
	})

	t.Run("existing questions, auto-assign", func(t *testing.T) {
		one := 1
		two := 2
		existing := []sdomain.ExamQuestion{
			{OrderIndex: &one},
			{OrderIndex: &two},
		}
		incoming := []*sdomain.ExamQuestion{
			{QuestionID: uuid.New()},
		}
		err := uc.validateAndAssignOrderIndexes(existing, incoming)
		assert.NoError(t, err)
		assert.Equal(t, 3, *incoming[0].OrderIndex)
	})

	t.Run("duplicate index", func(t *testing.T) {
		one := 1
		existing := []sdomain.ExamQuestion{
			{OrderIndex: &one},
		}
		incoming := []*sdomain.ExamQuestion{
			{OrderIndex: &one},
		}
		err := uc.validateAndAssignOrderIndexes(existing, incoming)
		assert.Error(t, err)
	})

	t.Run("gap detected", func(t *testing.T) {
		four := 4
		incoming := []*sdomain.ExamQuestion{
			{OrderIndex: &four},
		}
		err := uc.validateAndAssignOrderIndexes(nil, incoming)
		assert.Error(t, err)
	})
}

func TestRestoreExam(t *testing.T) {
	// Stub RunInTx to bypass live db pool
	oldRunInTx := RunInTx
	defer func() { RunInTx = oldRunInTx }()
	RunInTx = func(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
		return fn(nil)
	}

	t.Run("restore draft archived exam (ScheduledStart is nil)", func(t *testing.T) {
		examID := uuid.New()
		enterpriseID := uuid.New()
		exam := &sdomain.Exam{
			ID:             examID,
			EnterpriseID:   enterpriseID,
			Status:         sdomain.ExamArchived,
			ScheduledStart: nil,
		}

		var updatedExam *sdomain.Exam
		repo := &mockExamRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID, entID uuid.UUID) (*sdomain.Exam, error) {
				assert.Equal(t, examID, id)
				assert.Equal(t, enterpriseID, entID)
				return exam, nil
			},
			UpdateFunc: func(ctx context.Context, e *sdomain.Exam) error {
				updatedExam = e
				return nil
			},
		}

		uc := &examUsecase{
			examRepo: repo,
		}

		err := uc.RestoreExam(context.Background(), examID, enterpriseID)
		assert.NoError(t, err)
		assert.NotNil(t, updatedExam)
		assert.Equal(t, sdomain.ExamDraft, updatedExam.Status)
	})

	t.Run("restore scheduled/closed archived exam (ScheduledStart is set)", func(t *testing.T) {
		examID := uuid.New()
		enterpriseID := uuid.New()
		startTime := time.Now().Add(24 * time.Hour)
		exam := &sdomain.Exam{
			ID:             examID,
			EnterpriseID:   enterpriseID,
			Status:         sdomain.ExamArchived,
			ScheduledStart: &startTime,
		}

		var updatedExam *sdomain.Exam
		repo := &mockExamRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID, entID uuid.UUID) (*sdomain.Exam, error) {
				return exam, nil
			},
			UpdateFunc: func(ctx context.Context, e *sdomain.Exam) error {
				updatedExam = e
				return nil
			},
		}

		uc := &examUsecase{
			examRepo: repo,
		}

		err := uc.RestoreExam(context.Background(), examID, enterpriseID)
		assert.NoError(t, err)
		assert.NotNil(t, updatedExam)
		assert.Equal(t, sdomain.ExamClosed, updatedExam.Status)
	})

	t.Run("fail if exam is not archived", func(t *testing.T) {
		examID := uuid.New()
		enterpriseID := uuid.New()
		exam := &sdomain.Exam{
			ID:           examID,
			EnterpriseID: enterpriseID,
			Status:       sdomain.ExamDraft,
		}

		repo := &mockExamRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID, entID uuid.UUID) (*sdomain.Exam, error) {
				return exam, nil
			},
		}

		uc := &examUsecase{
			examRepo: repo,
		}

		err := uc.RestoreExam(context.Background(), examID, enterpriseID)
		assert.ErrorIs(t, err, domain.ErrInvalidStatus)
	})

	t.Run("fail if exam not found", func(t *testing.T) {
		examID := uuid.New()
		enterpriseID := uuid.New()

		repo := &mockExamRepository{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID, entID uuid.UUID) (*sdomain.Exam, error) {
				return nil, domain.ErrExamNotFound
			},
		}

		uc := &examUsecase{
			examRepo: repo,
		}

		err := uc.RestoreExam(context.Background(), examID, enterpriseID)
		assert.ErrorIs(t, err, domain.ErrExamNotFound)
	})
}
