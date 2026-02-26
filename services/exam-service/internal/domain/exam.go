package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ExamStatus string

const (
	ExamDraft     ExamStatus = "Draft"
	ExamScheduled ExamStatus = "Scheduled"
	ExamActive    ExamStatus = "Active"
	ExamClosed    ExamStatus = "Closed"
	ExamArchived  ExamStatus = "Archived"
)

type ExamQuestion struct {
	ID             uuid.UUID `db:"id" json:"id"`
	ExamID         uuid.UUID `db:"exam_id" json:"examId"`
	QuestionID     uuid.UUID `db:"question_id" json:"questionId"`
	PointsOverride *int      `db:"points_override" json:"pointsOverride,omitempty"`
	OrderIndex     *int      `db:"order_index" json:"orderIndex,omitempty"`

	Question *Question `json:"question,omitempty"` // populated if joined
}

type ExamRandomizationRule struct {
	ID            uuid.UUID        `db:"id" json:"id"`
	ExamID        uuid.UUID        `db:"exam_id" json:"examId"`
	Topic         *string          `db:"topic" json:"topic,omitempty"`
	Difficulty    *DifficultyLevel `db:"difficulty" json:"difficulty,omitempty"`
	QuestionCount int              `db:"question_count" json:"questionCount"`
}

type Exam struct {
	ID                  uuid.UUID              `db:"id" json:"id"`
	EnterpriseID        uuid.UUID              `db:"enterprise_id" json:"enterpriseId"`
	Title               string                 `db:"title" json:"title"`
	Description         *string                `db:"description" json:"description,omitempty"`
	DurationMinutes     int                    `db:"duration_minutes" json:"durationMinutes"`
	PassingScorePercent float64                `db:"passing_score_percent" json:"passingScorePercent"`
	NegativeMarking     bool                   `db:"negative_marking" json:"negativeMarking"`
	MaxParticipants     *int                   `db:"max_participants" json:"maxParticipants,omitempty"`
	InvitationMethod    string                 `db:"invitation_method" json:"invitationMethod"` // Email, Link, Token
	Status              ExamStatus             `db:"status" json:"status"`
	TemplateSourceID    *uuid.UUID             `db:"template_source_id" json:"templateSourceId,omitempty"`
	ScheduledStart      *time.Time             `db:"scheduled_start" json:"scheduledStart,omitempty"`
	ScheduledEnd        *time.Time             `db:"scheduled_end" json:"scheduledEnd,omitempty"`
	Settings            map[string]interface{} `db:"settings" json:"settings,omitempty"`
	CreatedBy           uuid.UUID              `db:"created_by" json:"createdBy"`
	CreatedAt           time.Time              `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time              `db:"updated_at" json:"updatedAt"`

	// Relational data
	Questions          []ExamQuestion          `json:"questions,omitempty"`
	RandomizationRules []ExamRandomizationRule `json:"randomizationRules,omitempty"`
}

type ExamRepository interface {
	Create(ctx context.Context, exam *Exam) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*Exam, error)
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*Exam, error)
	Update(ctx context.Context, exam *Exam) error
	Delete(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error

	AddQuestion(ctx context.Context, examID uuid.UUID, eq *ExamQuestion) error
	RemoveQuestion(ctx context.Context, examID uuid.UUID, questionID uuid.UUID) error
	UpdateQuestionMapping(ctx context.Context, examID uuid.UUID, eq *ExamQuestion) error

	AddRandomizationRule(ctx context.Context, examID uuid.UUID, rule *ExamRandomizationRule) error
	UpdateRandomizationRule(ctx context.Context, examID uuid.UUID, rule *ExamRandomizationRule) error
	DeleteRandomizationRule(ctx context.Context, examID uuid.UUID, ruleID uuid.UUID) error
}

type ExamUsecase interface {
	CreateExam(ctx context.Context, exam *Exam, userID uuid.UUID) (*Exam, error)
	GetExams(ctx context.Context, enterpriseID uuid.UUID) ([]*Exam, error)
	GetExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*Exam, error)
	UpdateExam(ctx context.Context, exam *Exam, userID uuid.UUID) error
	ScheduleExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, startTime time.Time, endTime time.Time, userID uuid.UUID) error
	CloneExam(ctx context.Context, sourceID uuid.UUID, enterpriseID uuid.UUID, cloneTitle string, userID uuid.UUID) (*Exam, error)
	PublishExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	CloseExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	DeleteExam(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error

	AddQuestionToExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) (*ExamQuestion, error)
	RemoveQuestionFromExam(ctx context.Context, enterpriseID, examID, questionID uuid.UUID) error
	UpdateExamQuestion(ctx context.Context, enterpriseID, examID, questionID uuid.UUID, pointsOverride *int, orderIndex *int) error

	AddRandomizationRule(ctx context.Context, enterpriseID, examID uuid.UUID, topic *string, difficulty *DifficultyLevel, questionCount int) (*ExamRandomizationRule, error)
	UpdateRandomizationRule(ctx context.Context, enterpriseID, examID, ruleID uuid.UUID, topic *string, difficulty *DifficultyLevel, questionCount int) error
	DeleteRandomizationRule(ctx context.Context, enterpriseID, examID, ruleID uuid.UUID) error
}
