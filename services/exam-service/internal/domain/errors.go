package domain

import "errors"

var (
	ErrQuestionNotFound    = errors.New("question not found")
	ErrExamNotFound        = errors.New("exam not found")
	ErrInvalidStatus       = errors.New("invalid exam status transition")
	ErrInvalidQuestion     = errors.New("invalid question data")
	ErrUnauthorized        = errors.New("unauthorized action")
	ErrDuplicateOrderIndex      = errors.New("duplicate order index")
	ErrInvalidOrderIndex        = errors.New("invalid order index: must start from 1")
	ErrOrderIndexGap            = errors.New("order index gap detected")
	ErrMappingNotFound          = errors.New("exam-question mapping not found")
	ErrNoQuestions              = errors.New("exam must have at least one question")
	ErrQuestionValidationFailed = errors.New("failed to validate question")
	ErrMarshalFailed            = errors.New("failed to marshal data")
	ErrInternal                 = errors.New("internal service error")
	ErrInsufficientTime         = errors.New("scheduled duration is less than the exam duration")
	ErrExamCannotBeDeleted      = errors.New("exam cannot be deleted because it is active, closed or archived")
)
