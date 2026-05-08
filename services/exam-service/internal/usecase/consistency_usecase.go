package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"go.uber.org/zap"
)

type consistencyUseCase struct {
	examRepo       domain.ExamRepository
	eventPublisher domain.EventPublisher
	log            *zap.Logger
}

// NewConsistencyUseCase creates a new ConsistencyUseCase that operates
// directly on the repository layer, bypassing normal usecase status guards.
// This mirrors the pattern used by auth-service and candidate-service.
func NewConsistencyUseCase(
	examRepo domain.ExamRepository,
	eventPublisher domain.EventPublisher,
	log *zap.Logger,
) domain.ConsistencyUseCase {
	return &consistencyUseCase{
		examRepo:       examRepo,
		eventPublisher: eventPublisher,
		log:            log.Named("consistency_usecase"),
	}
}

// HandleEnterpriseSuspended force-closes all Active exams (emitting
// exam.exam.closed for downstream consumers) and silently reverts all
// Scheduled exams to Draft.
func (uc *consistencyUseCase) HandleEnterpriseSuspended(ctx context.Context, enterpriseID uuid.UUID) error {
	uc.log.Info("handling enterprise suspended event", zap.String("enterprise_id", enterpriseID.String()))

	if err := uc.forceCloseActiveExams(ctx, enterpriseID); err != nil {
		return err
	}
	if err := uc.cancelScheduledExams(ctx, enterpriseID); err != nil {
		return err
	}
	return nil
}

// HandleEnterpriseHardDeleted performs the same actions as
// HandleEnterpriseSuspended and additionally archives any Closed exams so
// that no exam data remains in a live state after a permanent enterprise removal.
func (uc *consistencyUseCase) HandleEnterpriseHardDeleted(ctx context.Context, enterpriseID uuid.UUID) error {
	uc.log.Info("handling enterprise hard-deleted event", zap.String("enterprise_id", enterpriseID.String()))

	if err := uc.forceCloseActiveExams(ctx, enterpriseID); err != nil {
		return err
	}
	if err := uc.cancelScheduledExams(ctx, enterpriseID); err != nil {
		return err
	}
	if err := uc.archiveClosedExams(ctx, enterpriseID); err != nil {
		return err
	}
	return nil
}

// forceCloseActiveExams transitions all Active exams to Closed and publishes
// exam.exam.closed events so that the candidate-service can terminate any
// in-progress sessions.
func (uc *consistencyUseCase) forceCloseActiveExams(ctx context.Context, enterpriseID uuid.UUID) error {
	exams, err := uc.examRepo.GetByEnterpriseAndStatuses(ctx, enterpriseID, []sdomain.ExamStatus{sdomain.ExamActive})
	if err != nil {
		uc.log.Error("failed to fetch active exams for enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Error(err),
		)
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("force-closing active exams for suspended enterprise",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("count", len(exams)),
	)

	successCount := 0
	for _, exam := range exams {
		exam.Status = sdomain.ExamClosed
		if err := uc.examRepo.Update(ctx, exam); err != nil {
			uc.log.Error("failed to force-close active exam",
				zap.String("exam_id", exam.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Emit exam.exam.closed so candidate-service terminates in-progress sessions.
		// Fired as a goroutine (best-effort), matching the pattern in exam_usecase.go.
		e := exam // capture loop variable
		go func() {
			bgCtx := context.Background()
			if pubErr := uc.eventPublisher.PublishExamClosed(bgCtx, domain.ExamLifecycleEvent{
				ExamID:       e.ID,
				EnterpriseID: e.EnterpriseID,
				Title:        e.Title,
				StartTime:    e.ScheduledStart,
				EndTime:      e.ScheduledEnd,
			}); pubErr != nil {
				uc.log.Error("failed to publish ExamClosed event after force-close",
					zap.String("exam_id", e.ID.String()),
					zap.Error(pubErr),
				)
			}
		}()

		successCount++
	}

	uc.log.Info("completed force-closing active exams",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("failed", len(exams)-successCount),
	)
	return nil
}

// cancelScheduledExams silently reverts all Scheduled exams to Draft.
// No event is emitted because candidates have not yet been admitted into
// an active session, so no downstream action is required.
func (uc *consistencyUseCase) cancelScheduledExams(ctx context.Context, enterpriseID uuid.UUID) error {
	exams, err := uc.examRepo.GetByEnterpriseAndStatuses(ctx, enterpriseID, []sdomain.ExamStatus{sdomain.ExamScheduled})
	if err != nil {
		uc.log.Error("failed to fetch scheduled exams for enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Error(err),
		)
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("cancelling scheduled exams for suspended enterprise",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("count", len(exams)),
	)

	successCount := 0
	for _, exam := range exams {
		exam.Status = sdomain.ExamDraft
		if err := uc.examRepo.Update(ctx, exam); err != nil {
			uc.log.Error("failed to cancel scheduled exam",
				zap.String("exam_id", exam.ID.String()),
				zap.Error(err),
			)
			continue
		}
		successCount++
	}

	uc.log.Info("completed cancelling scheduled exams",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("failed", len(exams)-successCount),
	)
	return nil
}

// archiveClosedExams silently moves all Closed exams to Archived.
// Used only on hard-delete — no event is emitted.
func (uc *consistencyUseCase) archiveClosedExams(ctx context.Context, enterpriseID uuid.UUID) error {
	exams, err := uc.examRepo.GetByEnterpriseAndStatuses(ctx, enterpriseID, []sdomain.ExamStatus{sdomain.ExamClosed})
	if err != nil {
		uc.log.Error("failed to fetch closed exams for enterprise",
			zap.String("enterprise_id", enterpriseID.String()),
			zap.Error(err),
		)
		return err
	}

	if len(exams) == 0 {
		return nil
	}

	uc.log.Info("archiving closed exams for hard-deleted enterprise",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("count", len(exams)),
	)

	successCount := 0
	for _, exam := range exams {
		exam.Status = sdomain.ExamArchived
		if err := uc.examRepo.Update(ctx, exam); err != nil {
			uc.log.Error("failed to archive closed exam",
				zap.String("exam_id", exam.ID.String()),
				zap.Error(err),
			)
			continue
		}
		successCount++
	}

	uc.log.Info("completed archiving closed exams",
		zap.String("enterprise_id", enterpriseID.String()),
		zap.Int("total", len(exams)),
		zap.Int("success", successCount),
		zap.Int("failed", len(exams)-successCount),
	)
	return nil
}
