package sdomain

import (
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

	Question *Question `json:"question,omitempty"`
}

type ExamQuestionInput struct {
	QuestionID     uuid.UUID
	PointsOverride *int
	OrderIndex     *int
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
	Status              ExamStatus             `db:"status" json:"status"`
	TemplateSourceID    *uuid.UUID             `db:"template_source_id" json:"templateSourceId,omitempty"`
	ScheduledStart      *time.Time             `db:"scheduled_start" json:"scheduledStart,omitempty"`
	ScheduledEnd        *time.Time             `db:"scheduled_end" json:"scheduledEnd,omitempty"`
	Settings            map[string]any `db:"settings" json:"settings,omitempty"`
	CreatedBy           uuid.UUID              `db:"created_by" json:"createdBy"`
	CreatedAt           time.Time              `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time              `db:"updated_at" json:"updatedAt"`

	Questions          []ExamQuestion          `json:"questions,omitempty"`
	RandomizationRules []ExamRandomizationRule `json:"randomizationRules,omitempty"`
}
