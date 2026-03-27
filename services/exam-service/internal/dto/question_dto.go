package dto

import (
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
)

// QuestionOptionDTO defines the payload for question options.
type QuestionOptionDTO struct {
	Content   string `json:"content" binding:"required"`
	IsCorrect bool   `json:"isCorrect"`
}

// CreateQuestionRequest defines the payload for creating a new question.
type CreateQuestionRequest struct {
	Type           sdomain.QuestionType    `json:"type" binding:"required"`
	Topic          string                  `json:"topic" binding:"required"`
	Difficulty     sdomain.DifficultyLevel `json:"difficulty" binding:"required"`
	Title          string                  `json:"title" binding:"required"`
	Content        string                  `json:"content" binding:"required"`
	MediaURL       *string                 `json:"mediaUrl,omitempty"`
	Points         int                     `json:"points" binding:"required"`
	NegativePoints float64                 `json:"negativePoints"`
	Metadata       map[string]any          `json:"metadata,omitempty"`
	IsActive       bool                    `json:"isActive"`
	Options        []QuestionOptionDTO     `json:"options,omitempty"`
}

// UpdateQuestionRequest defines the payload for updating an existing question.
type UpdateQuestionRequest struct {
	Type           sdomain.QuestionType    `json:"type"`
	Topic          string                  `json:"topic"`
	Difficulty     sdomain.DifficultyLevel `json:"difficulty"`
	Title          string                  `json:"title"`
	Content        string                  `json:"content"`
	MediaURL       *string                 `json:"mediaUrl,omitempty"`
	Points         int                     `json:"points"`
	NegativePoints float64                 `json:"negativePoints"`
	Metadata       map[string]any          `json:"metadata,omitempty"`
	IsActive       bool                    `json:"isActive"`
	Options        []QuestionOptionDTO     `json:"options,omitempty"`
}
