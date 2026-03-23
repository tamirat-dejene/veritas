package dto

import (
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

// CreateExamRequest defines the payload for creating a new exam.
type CreateExamRequest struct {
	Title               string                 `json:"title" binding:"required"`
	Description         *string                `json:"description,omitempty"`
	DurationMinutes     int                    `json:"durationMinutes" binding:"required"`
	PassingScorePercent float64                `json:"passingScorePercent" binding:"required"`
	NegativeMarking     bool                   `json:"negativeMarking"`
	MaxParticipants     *int                   `json:"maxParticipants,omitempty"`
	TemplateSourceID    *uuid.UUID             `json:"templateSourceId,omitempty"`
	Settings            map[string]interface{} `json:"settings,omitempty"`
}

// UpdateExamRequest defines the payload for updating an existing exam.
type UpdateExamRequest struct {
	Title               string                 `json:"title"`
	Description         *string                `json:"description,omitempty"`
	DurationMinutes     int                    `json:"durationMinutes"`
	PassingScorePercent float64                `json:"passingScorePercent"`
	NegativeMarking     bool                   `json:"negativeMarking"`
	MaxParticipants     *int                   `json:"maxParticipants,omitempty"`
	Settings            map[string]interface{} `json:"settings,omitempty"`
}

// ScheduleExamRequest is the request body for scheduling an exam.
type ScheduleExamRequest struct {
	StartTime string `json:"startTime" binding:"required" example:"2026-03-01T10:00:00Z"`
	EndTime   string `json:"endTime" binding:"required" example:"2026-03-01T12:00:00Z"`
}

// CloneExamRequest is the request body for cloning an exam.
type CloneExamRequest struct {
	Title string `json:"title" binding:"required"`
}

// AddExamQuestionRequest is the request body for adding a question to an exam.
type AddExamQuestionRequest struct {
	QuestionID     string `json:"questionId" binding:"required"`
	PointsOverride *int   `json:"pointsOverride,omitempty"`
	OrderIndex     *int   `json:"orderIndex,omitempty"`
}

// AddExamQuestionsBulkRequest is the request body for adding multiple questions to an exam.
type AddExamQuestionsBulkRequest struct {
	Questions []AddExamQuestionRequest `json:"questions" binding:"required,min=1,dive"`
}

// UpdateExamQuestionRequest is the request body for updating exam-question mapping.
type UpdateExamQuestionRequest struct {
	PointsOverride *int `json:"pointsOverride,omitempty"`
	OrderIndex     *int `json:"orderIndex,omitempty"`
}

// ExamRuleRequest is the request body for exam randomization rules.
type ExamRuleRequest struct {
	Topic         *string                 `json:"topic,omitempty"`
	Difficulty    *domain.DifficultyLevel `json:"difficulty,omitempty"`
	QuestionCount int                     `json:"questionCount" binding:"required"`
}
