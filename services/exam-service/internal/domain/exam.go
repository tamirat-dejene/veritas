package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type ExamRepository interface {
	Create(ctx context.Context, exam *sdomain.Exam) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error)
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Exam], error)
	Update(ctx context.Context, exam *sdomain.Exam) error
	Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	CountByEnterpriseAndStatus(ctx context.Context, enterpriseID uuid.UUID, status sdomain.ExamStatus) (int, error)

	AddQuestions(ctx context.Context, examID uuid.UUID, eqs []*sdomain.ExamQuestion) error
	GetExamQuestions(ctx context.Context, examID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.ExamQuestion], error)
	RemoveQuestion(ctx context.Context, examID uuid.UUID, questionID uuid.UUID) error
	UpdateQuestionMapping(ctx context.Context, examID uuid.UUID, eq *sdomain.ExamQuestion) error

	WithTx(tx pgx.Tx) ExamRepository
}

type ExamUsecase interface {
	CreateExam(ctx context.Context, exam *sdomain.Exam, userID uuid.UUID) (*sdomain.Exam, error)
	GetExams(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) (pagination.PaginatedResponse[*sdomain.Exam], error)
	GetExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*sdomain.Exam, error)
	UpdateExam(ctx context.Context, exam *sdomain.Exam, userID uuid.UUID) error
	ScheduleExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, startTime time.Time, endTime time.Time, userID uuid.UUID) error
	CloneExam(ctx context.Context, sourceID uuid.UUID, enterpriseID uuid.UUID, cloneTitle string, userID uuid.UUID) (*sdomain.Exam, error)
	PublishExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	CloseExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	DeleteExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	GetActiveExamsCount(ctx context.Context, enterpriseID uuid.UUID) (int, error)

	AddQuestionsToExam(ctx context.Context, enterpriseID, examID uuid.UUID, inputs []sdomain.ExamQuestionInput) ([]*sdomain.ExamQuestion, error)
	GetExamQuestions(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params, withCorrectAnswer bool) (pagination.PaginatedResponse[*sdomain.ExamQuestion], error)
	RemoveQuestionFromExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID) error
	UpdateExamQuestion(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) error

}
