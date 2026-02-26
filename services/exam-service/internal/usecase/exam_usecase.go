package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type examUsecase struct {
	examRepo     domain.ExamRepository
	questionRepo domain.QuestionRepository
}

func NewExamUsecase(examRepo domain.ExamRepository, questionRepo domain.QuestionRepository) domain.ExamUsecase {
	return &examUsecase{
		examRepo:     examRepo,
		questionRepo: questionRepo,
	}
}

func (uc *examUsecase) CreateExam(ctx context.Context, exam *domain.Exam, userID uuid.UUID) (*domain.Exam, error) {
	exam.CreatedBy = userID
	exam.Status = domain.ExamDraft

	if err := uc.examRepo.Create(ctx, exam); err != nil {
		return nil, err
	}

	return exam, nil
}

func (uc *examUsecase) UpdateExam(ctx context.Context, exam *domain.Exam, userID uuid.UUID) error {
	existing, err := uc.examRepo.GetByID(ctx, exam.ID, exam.EnterpriseID)
	if err != nil {
		return err
	}

	// Only draft exams can be fully updated
	if existing.Status != domain.ExamDraft {
		// Just an example check, we can allow certain updates if Scheduled
		if existing.Status == domain.ExamActive || existing.Status == domain.ExamClosed || existing.Status == domain.ExamArchived {
			return domain.ErrInvalidStatus
		}
	}

	return uc.examRepo.Update(ctx, exam)
}

func (uc *examUsecase) ScheduleExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, startTime time.Time, endTime time.Time, userID uuid.UUID) error {
	exam, err := uc.examRepo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return err
	}

	if exam.Status != domain.ExamDraft && exam.Status != domain.ExamScheduled {
		return domain.ErrInvalidStatus
	}

	exam.Status = domain.ExamScheduled
	exam.ScheduledStart = &startTime
	exam.ScheduledEnd = &endTime

	return uc.examRepo.Update(ctx, exam)
}

func (uc *examUsecase) CloneExam(ctx context.Context, sourceID uuid.UUID, enterpriseID uuid.UUID, cloneTitle string, userID uuid.UUID) (*domain.Exam, error) {
	source, err := uc.examRepo.GetByID(ctx, sourceID, enterpriseID)
	if err != nil {
		return nil, err
	}

	clone := &domain.Exam{
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

	err = uc.examRepo.Create(ctx, clone)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloned exam: %w", err)
	}

	return clone, nil
}
