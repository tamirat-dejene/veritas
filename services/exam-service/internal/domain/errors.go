package domain

import "errors"

var (
	ErrQuestionNotFound    = errors.New("question not found")
	ErrExamNotFound        = errors.New("exam not found")
	ErrInvalidStatus       = errors.New("invalid exam status transition")
	ErrInvalidQuestion     = errors.New("invalid question data")
	ErrUnauthorized        = errors.New("unauthorized action")
	ErrDuplicateOrderIndex = errors.New("duplicate order index")
	ErrInvalidOrderIndex   = errors.New("invalid order index: must start from 1")
	ErrOrderIndexGap       = errors.New("order index gap detected")
)
