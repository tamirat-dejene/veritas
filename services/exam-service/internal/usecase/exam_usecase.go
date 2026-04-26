package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type examUsecase struct {
	pool         *pgxpool.Pool
	examRepo     domain.ExamRepository
	questionRepo domain.QuestionRepository
}

func NewExamUsecase(pool *pgxpool.Pool, examRepo domain.ExamRepository, questionRepo domain.QuestionRepository) domain.ExamUsecase {
	return &examUsecase{
		pool:         pool,
		examRepo:     examRepo,
		questionRepo: questionRepo,
	}
}

func (uc *examUsecase) CreateExam(ctx context.Context, exam *sdomain.Exam, userID uuid.UUID) (*sdomain.Exam, error) {
	exam.CreatedBy = userID
	exam.Status = sdomain.ExamDraft

	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		// Convert pointer array to struct array for helper (or modify helper)
		var incoming []*sdomain.ExamQuestion
		for i := range exam.Questions {
			incoming = append(incoming, &exam.Questions[i])
		}
		if err := uc.validateAndAssignOrderIndexes(nil, incoming); err != nil {
			return err
		}

		return uc.examRepo.WithTx(tx).Create(ctx, exam)
	})
	if err != nil {
		return nil, err
	}

	return exam, nil
}

func (uc *examUsecase) UpdateExam(ctx context.Context, exam *sdomain.Exam, userID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		existing, err := uc.examRepo.WithTx(tx).GetByID(ctx, exam.ID, exam.EnterpriseID)
		if err != nil {
			return err
		}

		// Only draft exams can be fully updated
		if existing.Status != sdomain.ExamDraft {
			if existing.Status == sdomain.ExamActive || existing.Status == sdomain.ExamClosed || existing.Status == sdomain.ExamArchived {
				return domain.ErrInvalidStatus
			}
		}
		exam.Status = existing.Status

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) ScheduleExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, startTime time.Time, endTime time.Time, userID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		// Only draft exams can be scheduled
		if exam.Status != sdomain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		exam.Status = sdomain.ExamScheduled
		exam.ScheduledStart = &startTime
		exam.ScheduledEnd = &endTime

		// Scheduled duration must be at least as long as the exam's durationMinutes
		duration := endTime.Sub(startTime)
		if duration < time.Duration(exam.DurationMinutes)*time.Minute {
			return domain.ErrInsufficientTime
		}

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) CloneExam(ctx context.Context, sourceID uuid.UUID, enterpriseID uuid.UUID, cloneTitle string, userID uuid.UUID) (*sdomain.Exam, error) {
	var clone *sdomain.Exam
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		source, err := uc.examRepo.WithTx(tx).GetByID(ctx, sourceID, enterpriseID)
		if err != nil {
			return err
		}

		clone = &sdomain.Exam{
			EnterpriseID:        enterpriseID,
			Title:               cloneTitle,
			Description:         source.Description,
			DurationMinutes:     source.DurationMinutes,
			PassingScorePercent: source.PassingScorePercent,
			NegativeMarking:     source.NegativeMarking,
			MaxParticipants:     source.MaxParticipants,
			Status:              sdomain.ExamDraft,
			TemplateSourceID:    &source.ID,
			Settings:            source.Settings,
			CreatedBy:           userID,
		}

		// Clone mappings (but reset IDs)
		for _, q := range source.Questions {
			qClone := q
			qClone.ID = uuid.Nil
			qClone.ExamID = uuid.Nil
			clone.Questions = append(clone.Questions, qClone)
		}

		err = uc.examRepo.WithTx(tx).Create(ctx, clone)
		if err != nil {
			return fmt.Errorf("%w: clone creation: %v", domain.ErrInternal, err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return clone, nil
}

func (uc *examUsecase) GetExams(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Exam], error) {
	return uc.examRepo.ListByEnterprise(ctx, enterpriseID, params)
}

func (uc *examUsecase) GetExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error) {
	return uc.examRepo.GetByID(ctx, id, enterpriseID)
}

func (uc *examUsecase) PublishExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		if exam.Status != sdomain.ExamDraft && exam.Status != sdomain.ExamScheduled {
			return domain.ErrInvalidStatus
		}

		// Verification logic
		if len(exam.Questions) == 0 {
			return domain.ErrNoQuestions
		}

		exam.Status = sdomain.ExamActive

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) GetExamQuestions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params, withCorrectAnswer bool) (pagination.PaginatedResponse[*sdomain.ExamQuestion], error) {
	_, err := uc.examRepo.GetByID(ctx, examID, enterpriseID)
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
	}

	paginatedMappings, err := uc.examRepo.GetExamQuestions(ctx, examID, params)
	if err != nil {
		return pagination.PaginatedResponse[*sdomain.ExamQuestion]{}, err
	}

	var result []*sdomain.ExamQuestion
	for _, eq := range paginatedMappings.Data {
		q, err := uc.questionRepo.GetByID(ctx, eq.QuestionID, enterpriseID, withCorrectAnswer)
		if err == nil && q != nil {
			eqCopy := *eq
			eqCopy.Question = q
			result = append(result, &eqCopy)
		}
	}

	return pagination.NewPaginatedResponse(result, paginatedMappings.Metadata.TotalElements, params), nil
}

func (uc *examUsecase) CloseExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		if exam.Status != sdomain.ExamActive {
			return domain.ErrInvalidStatus
		}

		exam.Status = sdomain.ExamClosed

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) DeleteExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return uc.examRepo.Delete(ctx, id, enterpriseID)
}

func (uc *examUsecase) AddQuestionsToExam(ctx context.Context, enterpriseID, examID uuid.UUID, inputs []sdomain.ExamQuestionInput) ([]*sdomain.ExamQuestion, error) {
	var eqs []*sdomain.ExamQuestion
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != sdomain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		for _, input := range inputs {
			_, err = uc.questionRepo.WithTx(tx).GetByID(ctx, input.QuestionID, enterpriseID, false)
			if err != nil {
				return fmt.Errorf("%w: question %s: %v", domain.ErrQuestionValidationFailed, input.QuestionID, err)
			}

			eq := &sdomain.ExamQuestion{
				ExamID:         examID,
				QuestionID:     input.QuestionID,
				PointsOverride: input.PointsOverride,
				OrderIndex:     input.OrderIndex,
			}
			eqs = append(eqs, eq)
		}

		if err := uc.validateAndAssignOrderIndexes(exam.Questions, eqs); err != nil {
			return err
		}

		err = uc.examRepo.WithTx(tx).AddQuestions(ctx, examID, eqs)
		if err != nil {
			return fmt.Errorf("%w: add questions: %v", domain.ErrInternal, err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return eqs, nil
}

func (uc *examUsecase) RemoveQuestionFromExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != sdomain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		// Find the index of the question being removed
		var removedIdx int
		found := false
		for _, eq := range exam.Questions {
			if eq.QuestionID == questionID {
				if eq.OrderIndex != nil {
					removedIdx = *eq.OrderIndex
					found = true
				}
				break
			}
		}

		if err := uc.examRepo.WithTx(tx).RemoveQuestion(ctx, examID, questionID); err != nil {
			return err
		}

		// If it had an index, shift all subsequent questions
		if found {
			for _, eq := range exam.Questions {
				if eq.OrderIndex != nil && *eq.OrderIndex > removedIdx {
					newIdx := *eq.OrderIndex - 1
					eq.OrderIndex = &newIdx
					if err := uc.examRepo.WithTx(tx).UpdateQuestionMapping(ctx, examID, &eq); err != nil {
						return fmt.Errorf("%w: shift index %s: %v", domain.ErrInternal, eq.QuestionID, err)
					}
				}
			}
		}
		return nil
	})
}

func (uc *examUsecase) UpdateExamQuestion(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != sdomain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		// If OrderIndex is being updated, handle reordering
		if orderIndex != nil {
			total := len(exam.Questions)
			if *orderIndex <= 0 || *orderIndex > total {
				return domain.ErrInvalidOrderIndex
			}

			var oldIndex int
			found := false
			for _, q := range exam.Questions {
				if q.QuestionID == questionID {
					if q.OrderIndex != nil {
						oldIndex = *q.OrderIndex
						found = true
					}
					break
				}
			}

			if found && oldIndex != *orderIndex {
				if *orderIndex > oldIndex {
					// Moving up: shift questions between (oldIndex, newIndex] down by 1
					for _, q := range exam.Questions {
						if q.OrderIndex != nil && *q.OrderIndex > oldIndex && *q.OrderIndex <= *orderIndex && q.QuestionID != questionID {
							newVal := *q.OrderIndex - 1
							q.OrderIndex = &newVal
							if err := uc.examRepo.WithTx(tx).UpdateQuestionMapping(ctx, examID, &q); err != nil {
								return err
							}
						}
					}
				} else {
					// Moving down: shift questions between [newIndex, oldIndex) up by 1
					for _, q := range exam.Questions {
						if q.OrderIndex != nil && *q.OrderIndex >= *orderIndex && *q.OrderIndex < oldIndex && q.QuestionID != questionID {
							newVal := *q.OrderIndex + 1
							q.OrderIndex = &newVal
							if err := uc.examRepo.WithTx(tx).UpdateQuestionMapping(ctx, examID, &q); err != nil {
								return err
							}
						}
					}
				}
			}
		}

		eq := &sdomain.ExamQuestion{
			ExamID:         examID,
			QuestionID:     questionID,
			PointsOverride: pointsOverride,
			OrderIndex:     orderIndex,
		}

		return uc.examRepo.WithTx(tx).UpdateQuestionMapping(ctx, examID, eq)
	})
}

func (uc *examUsecase) validateAndAssignOrderIndexes(existing []sdomain.ExamQuestion, incoming []*sdomain.ExamQuestion) error {
	total := len(existing) + len(incoming)
	used := make(map[int]bool)

	for _, eq := range existing {
		if eq.OrderIndex != nil {
			if *eq.OrderIndex <= 0 {
				return domain.ErrInvalidOrderIndex
			}
			if used[*eq.OrderIndex] {
				return domain.ErrDuplicateOrderIndex
			}
			used[*eq.OrderIndex] = true
		}
	}

	// First pass: validate provided indexes for incoming questions
	for _, eq := range incoming {
		if eq.OrderIndex != nil {
			if *eq.OrderIndex <= 0 {
				return domain.ErrInvalidOrderIndex
			}
			if *eq.OrderIndex > total {
				return domain.ErrOrderIndexGap
			}
			if used[*eq.OrderIndex] {
				return domain.ErrDuplicateOrderIndex
			}
			used[*eq.OrderIndex] = true
		}
	}

	// Second pass: auto-assign missing indexes
	for _, eq := range incoming {
		if eq.OrderIndex == nil {
			for j := 1; j <= total; j++ {
				if !used[j] {
					idx := j
					eq.OrderIndex = &idx
					used[j] = true
					break
				}
			}
		}
	}

	// Final gap check
	for i := 1; i <= total; i++ {
		if !used[i] {
			return domain.ErrOrderIndexGap
		}
	}

	return nil
}
