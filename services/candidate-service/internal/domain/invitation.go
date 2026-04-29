package domain

import (
	"time"

	"github.com/google/uuid"
)

// EnrollmentResult is returned by EnrollCandidates for each enrolled candidate.
// It contains the invitation URL (with opaque code) that the admin can copy and
// distribute manually, or later trigger via NotifyCandidates.
// The raw JWT and raw opaque code are NEVER included here.
type EnrollmentResult struct {
	EnrollmentID  uuid.UUID        `json:"enrollmentId"`
	CandidateID   uuid.UUID        `json:"candidateId"`
	InvitationURL string           `json:"invitationUrl"` // {PORTAL_BASE_URL}/exam/start?code={opaque}
	Status        EnrollmentStatus `json:"status"`        // always "Pending" at enroll time
}

// NotifyResult is returned by NotifyCandidates / NotifyCandidate describing
// whether an email was sent for a given enrollment.
type NotifyResult struct {
	EnrollmentID uuid.UUID `json:"enrollmentId"`
	CandidateID  uuid.UUID `json:"candidateId"`
	// NotifyStatus is "sent" when an email was dispatched, or "skipped_no_email"
	// when the candidate profile has no email address.
	NotifyStatus string `json:"notifyStatus"`
}

// CandidateEnrollmentInvitedEvent is the Kafka payload published on topic
// candidate.enrollment.invited. The notification-service consumes this event
// and sends the invitation email to the candidate.
type CandidateEnrollmentInvitedEvent struct {
	EnrollmentID   uuid.UUID `json:"enrollment_id"`
	CandidateID    uuid.UUID `json:"candidate_id"`
	ExamID         uuid.UUID `json:"exam_id"`
	EnterpriseID   uuid.UUID `json:"enterprise_id"`
	CandidateName  string    `json:"candidate_name"`
	CandidateEmail string    `json:"candidate_email"`
	ExamTitle      string    `json:"exam_title"`
	// InvitationURL contains the opaque code in the query string — safe to email.
	// It NEVER contains the raw JWT.
	InvitationURL string    `json:"invitation_url"`
	ExpiresAt     time.Time `json:"expires_at"`
	Timestamp     int64     `json:"timestamp"`
}
