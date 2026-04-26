package usecase

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tamirat-dejene/veritas/shared/domain"
)

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
