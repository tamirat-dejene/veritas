package handler

import "github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"

// ErrorResponse is the standard error payload for this service.
type ErrorResponse struct {
	Error string `json:"error"`
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
