package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExamCreatedEvent is the payload for the exam.exam.created event.
type ExamCreatedEvent struct {
	ExamID       uuid.UUID `json:"exam_id"`
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Title        string    `json:"title"`
	AdminEmail   string    `json:"admin_email"`
	Timestamp    int64     `json:"timestamp"`
}

// ExamLifecycleEvent is the payload for scheduled, published, and closed events.
type ExamLifecycleEvent struct {
	ExamID          uuid.UUID `json:"exam_id"`
	EnterpriseID    uuid.UUID `json:"enterprise_id"`
	Title           string    `json:"title"`
	AdminEmail      string    `json:"admin_email"`
	CandidateEmails []string  `json:"candidate_emails"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Timestamp       int64     `json:"timestamp"`
}
