package domain

import "errors"

var (
	// Candidate constraints
	ErrCandidateNotFound     = errors.New("candidate not found")
	ErrDuplicateExternalID   = errors.New("a candidate with this external ID already exists")
	ErrUnauthorizedCandidate = errors.New("unauthorized candidate access")

	// Enrollment constraints
	ErrEnrollmentNotFound = errors.New("enrollment not found")
	ErrMaxAttemptsReached = errors.New("max exam attempts reached")

	// Session constraints
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionNotActive        = errors.New("session is not active")
	ErrSessionTerminated       = errors.New("session has been terminated")
	ErrSessionAlreadySubmitted = errors.New("session has already been submitted")
	ErrSessionAlreadyActive    = errors.New("an active session already exists for this enrollment")
	ErrSessionExpired          = errors.New("session has expired")
	ErrExamNotScheduled        = errors.New("exam is not currently within its scheduled window")
	ErrInvalidExamStatus       = errors.New("exam should be Scheduled to enroll candidates")
	ErrInvalidAccessToken      = errors.New("invalid or expired access token")
	ErrQuestionNotFound        = errors.New("question not found in session snapshot")
	ErrInvalidAnswerFormat     = errors.New("invalid answer format for question type")
	ErrSubmissionExists        = errors.New("a submission already exists for this session")
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrUnauthorizedAccess      = errors.New("unauthorized to access this resource")
	ErrInvalidEnrollmentTime   = errors.New("invalid enrollment time")

	// Standardized System & Validation errors
	ErrEnterpriseIDMissing = errors.New("enterprise ID missing")
	ErrEnrollmentIDMissing = errors.New("enrollment ID mapping missing")
	ErrCandidateIDMissing  = errors.New("candidate mapping missing")
	ErrInvalidIDFormat     = errors.New("invalid ID format")
	ErrMissingFile         = errors.New("missing file field in request")
	ErrFileTooLarge        = errors.New("file size exceeds limit")
	ErrReadFailed          = errors.New("failed to read data")
	ErrNoValidCandidates   = errors.New("no valid candidates found in CSV")
	ErrInternal            = errors.New("internal server error")
	ErrUnauthorizedContext = errors.New("unauthorized context")
	ErrInvalidToken        = errors.New("invalid token")
	ErrNotAString          = errors.New("value is not a string")
	ErrNotSupported        = errors.New("operation not supported")
)
