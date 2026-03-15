package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
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

func (uc *examUsecase) CreateExam(ctx context.Context, exam *domain.Exam, userID uuid.UUID) (*domain.Exam, error) {
	exam.CreatedBy = userID
	exam.Status = domain.ExamDraft

	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		return uc.examRepo.WithTx(tx).Create(ctx, exam)
	})
	if err != nil {
		return nil, err
	}

	return exam, nil
}

func (uc *examUsecase) UpdateExam(ctx context.Context, exam *domain.Exam, userID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		existing, err := uc.examRepo.WithTx(tx).GetByID(ctx, exam.ID, exam.EnterpriseID)
		if err != nil {
			return err
		}

		// Only draft exams can be fully updated
		if existing.Status != domain.ExamDraft {
			if existing.Status == domain.ExamActive || existing.Status == domain.ExamClosed || existing.Status == domain.ExamArchived {
				return domain.ErrInvalidStatus
			}
		}

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) ScheduleExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, startTime time.Time, endTime time.Time, userID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		if exam.Status != domain.ExamDraft && exam.Status != domain.ExamScheduled {
			return domain.ErrInvalidStatus
		}

		exam.Status = domain.ExamScheduled
		exam.ScheduledStart = &startTime
		exam.ScheduledEnd = &endTime

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) CloneExam(ctx context.Context, sourceID uuid.UUID, enterpriseID uuid.UUID, cloneTitle string, userID uuid.UUID) (*domain.Exam, error) {
	var clone *domain.Exam
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		source, err := uc.examRepo.WithTx(tx).GetByID(ctx, sourceID, enterpriseID)
		if err != nil {
			return err
		}

		clone = &domain.Exam{
			EnterpriseID:        enterpriseID,
			Title:               cloneTitle,
			Description:         source.Description,
			DurationMinutes:     source.DurationMinutes,
			PassingScorePercent: source.PassingScorePercent,
			NegativeMarking:     source.NegativeMarking,
			MaxParticipants:     source.MaxParticipants,
			InvitationMethod:    source.InvitationMethod,
			Status:              domain.ExamDraft,
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

		for _, r := range source.RandomizationRules {
			rClone := r
			rClone.ID = uuid.Nil
			rClone.ExamID = uuid.Nil
			clone.RandomizationRules = append(clone.RandomizationRules, rClone)
		}

		err = uc.examRepo.WithTx(tx).Create(ctx, clone)
		if err != nil {
			return fmt.Errorf("failed to create cloned exam: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return clone, nil
}

func (uc *examUsecase) GetExams(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.Exam, error) {
	return uc.examRepo.ListByEnterprise(ctx, enterpriseID)
}

func (uc *examUsecase) GetExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.Exam, error) {
	return uc.examRepo.GetByID(ctx, id, enterpriseID)
}

func (uc *examUsecase) PublishExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		if exam.Status != domain.ExamDraft && exam.Status != domain.ExamScheduled {
			return domain.ErrInvalidStatus
		}

		// Verification logic
		if len(exam.Questions) == 0 && len(exam.RandomizationRules) == 0 {
			return fmt.Errorf("exam must have at least one question or randomization rule to be published")
		}

		exam.Status = domain.ExamActive

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) GetExamQuestions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID) ([]*domain.ExamQuestion, error) {
	exam, err := uc.examRepo.GetByID(ctx, examID, enterpriseID)
	if err != nil {
		return nil, err
	}

	var result []*domain.ExamQuestion
	for _, eq := range exam.Questions {
		q, err := uc.questionRepo.GetByID(ctx, eq.QuestionID, enterpriseID)
		if err == nil && q != nil {
			eqCopy := eq
			eqCopy.Question = q
			result = append(result, &eqCopy)
		}
	}
	return result, nil
}

func (uc *examUsecase) CloseExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		if exam.Status != domain.ExamActive {
			return domain.ErrInvalidStatus
		}

		exam.Status = domain.ExamClosed

		return uc.examRepo.WithTx(tx).Update(ctx, exam)
	})
}

func (uc *examUsecase) DeleteExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	return uc.examRepo.Delete(ctx, id, enterpriseID)
}

func (uc *examUsecase) AddQuestionToExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) (*domain.ExamQuestion, error) {
	var eq *domain.ExamQuestion
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		_, err = uc.questionRepo.WithTx(tx).GetByID(ctx, questionID, enterpriseID)
		if err != nil {
			return err
		}

		eq = &domain.ExamQuestion{
			ExamID:         examID,
			QuestionID:     questionID,
			PointsOverride: pointsOverride,
			OrderIndex:     orderIndex,
		}

		err = uc.examRepo.WithTx(tx).AddQuestion(ctx, examID, eq)
		if err != nil {
			return fmt.Errorf("failed to add question to exam: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return eq, nil
}

func (uc *examUsecase) RemoveQuestionFromExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		return uc.examRepo.WithTx(tx).RemoveQuestion(ctx, examID, questionID)
	})
}

func (uc *examUsecase) UpdateExamQuestion(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		eq := &domain.ExamQuestion{
			ExamID:         examID,
			QuestionID:     questionID,
			PointsOverride: pointsOverride,
			OrderIndex:     orderIndex,
		}

		return uc.examRepo.WithTx(tx).UpdateQuestionMapping(ctx, examID, eq)
	})
}

func (uc *examUsecase) AddRandomizationRule(ctx context.Context, enterpriseID, examID uuid.UUID, topic *string, difficulty *domain.DifficultyLevel, questionCount int) (*domain.ExamRandomizationRule, error) {
	var rule *domain.ExamRandomizationRule
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		rule = &domain.ExamRandomizationRule{
			ExamID:        examID,
			Topic:         topic,
			Difficulty:    difficulty,
			QuestionCount: questionCount,
		}

		if err := uc.examRepo.WithTx(tx).AddRandomizationRule(ctx, examID, rule); err != nil {
			return fmt.Errorf("failed to add randomization rule: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return rule, nil
}

func (uc *examUsecase) UpdateRandomizationRule(ctx context.Context, enterpriseID, examID, ruleID uuid.UUID, topic *string, difficulty *domain.DifficultyLevel, questionCount int) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		rule := &domain.ExamRandomizationRule{
			ID:            ruleID,
			ExamID:        examID,
			Topic:         topic,
			Difficulty:    difficulty,
			QuestionCount: questionCount,
		}

		return uc.examRepo.WithTx(tx).UpdateRandomizationRule(ctx, examID, rule)
	})
}

func (uc *examUsecase) DeleteRandomizationRule(ctx context.Context, enterpriseID, examID, ruleID uuid.UUID) error {
	return RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		exam, err := uc.examRepo.WithTx(tx).GetByID(ctx, examID, enterpriseID)
		if err != nil {
			return err
		}
		if exam.Status != domain.ExamDraft {
			return domain.ErrInvalidStatus
		}

		return uc.examRepo.WithTx(tx).DeleteRandomizationRule(ctx, examID, ruleID)
	})
}
