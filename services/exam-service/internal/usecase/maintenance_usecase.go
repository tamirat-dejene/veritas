package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"go.uber.org/zap"
)

type maintenanceUseCase struct {
	examRepo domain.ExamRepository
	examUC   domain.ExamUsecase
	log      *zap.Logger
}

func NewMaintenanceUseCase(
	examRepo domain.ExamRepository,
	examUC domain.ExamUsecase,
	log *zap.Logger,
) domain.MaintenanceUseCase {
	return &maintenanceUseCase{
		examRepo: examRepo,
		examUC:   examUC,
		log:      log.Named("maintenance_usecase"),
	}
}

func (uc *maintenanceUseCase) PublishScheduledExams(ctx context.Context) error {
	batchSize := 100
	exams, err := uc.examRepo.GetScheduledExamsDue(ctx, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch scheduled exams due for publishing", zap.Error(err))
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("publishing scheduled exams", zap.Int("count", len(exams)))

	successCount := 0
	unscheduledCount := 0
	for _, exam := range exams {
		if err := uc.examUC.PublishExam(ctx, exam.ID, exam.EnterpriseID); err != nil {
			if errors.Is(err, domain.ErrNoQuestions) {
				if unschedErr := uc.examUC.UnscheduleExam(ctx, exam.ID, exam.EnterpriseID); unschedErr != nil {
					uc.log.Error("failed to unschedule unpublishable exam",
						zap.String("exam_id", exam.ID.String()),
						zap.Error(unschedErr))
				} else {
					uc.log.Warn("exam unscheduled due to missing questions; reverted to Draft",
						zap.String("exam_id", exam.ID.String()))
					unscheduledCount++
				}
			} else {
				uc.log.Error("failed to auto-publish exam",
					zap.String("exam_id", exam.ID.String()),
					zap.Error(err))
			}
			continue
		}
		successCount++
	}

	failedCount := len(exams) - successCount - unscheduledCount
	uc.log.Info("completed auto-publishing scheduled exams",
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("unscheduled", unscheduledCount),
		zap.Int("failed", failedCount))

	return nil
}

func (uc *maintenanceUseCase) CloseExpiredExams(ctx context.Context) error {
	batchSize := 100
	exams, err := uc.examRepo.GetActiveExamsPastEnd(ctx, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch active exams past end time", zap.Error(err))
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("closing expired active exams", zap.Int("count", len(exams)))

	successCount := 0
	for _, exam := range exams {
		if err := uc.examUC.CloseExam(ctx, exam.ID, exam.EnterpriseID); err != nil {
			uc.log.Error("failed to auto-close exam",
				zap.String("exam_id", exam.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	uc.log.Info("completed auto-closing expired exams",
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("failed", len(exams)-successCount))

	return nil
}

func (uc *maintenanceUseCase) ArchiveStaleExams(ctx context.Context) error {
	retentionDays := 30
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	batchSize := 100

	exams, err := uc.examRepo.GetStaleClosedExams(ctx, cutoff, batchSize)
	if err != nil {
		uc.log.Error("failed to fetch stale closed exams", zap.Error(err))
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("archiving stale closed exams", zap.Int("count", len(exams)))

	successCount := 0
	for _, exam := range exams {
		if err := uc.examUC.ArchiveExam(ctx, exam.ID, exam.EnterpriseID); err != nil {
			uc.log.Error("failed to auto-archive exam",
				zap.String("exam_id", exam.ID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	uc.log.Info("completed auto-archiving stale exams",
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("failed", len(exams)-successCount))

	return nil
}
