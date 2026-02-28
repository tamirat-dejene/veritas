package domain

import "errors"

var (
	// Candidate constraints
	ErrCandidateNotFound   = errors.New("candidate not found")
	ErrDuplicateExternalID = errors.New("a candidate with this external ID already exists")

	// Enrollment constraints
	ErrEnrollmentNotFound = errors.New("enrollment not found")
	ErrMaxAttemptsReached = errors.New("max exam attempts reached")

	// Session constraints
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionNotActive   = errors.New("session is not active")
	ErrSessionExpired     = errors.New("session has expired")
	ErrExamNotScheduled   = errors.New("exam is not currently within its scheduled window")
	ErrInvalidAccessToken = errors.New("invalid or expired access token")
	ErrQuestionNotFound   = errors.New("question not found in session snapshot")
	ErrSubmissionExists   = errors.New("a submission already exists for this session")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrUnauthorizedAccess = errors.New("unauthorized to access this resource")
)
