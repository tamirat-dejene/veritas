package domain

import "errors"

var (
	ErrQuestionNotFound = errors.New("question not found")
	ErrExamNotFound     = errors.New("exam not found")
	ErrInvalidStatus    = errors.New("invalid exam status transition")
	ErrInvalidQuestion  = errors.New("invalid question data")
	ErrUnauthorized     = errors.New("unauthorized action")
)
