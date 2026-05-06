package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// EnrollmentStatus tracks the invitation lifecycle of an enrollment.
type EnrollmentStatus string

const (
	StatusPending   EnrollmentStatus = "Pending"   // enrolled, admin has not yet sent notification
	StatusInvited   EnrollmentStatus = "Invited"   // notification email sent to candidate
	StatusOpened    EnrollmentStatus = "Opened"    // candidate redeemed the invitation link
	StatusStarted   EnrollmentStatus = "Started"   // candidate has an active exam session
	StatusCompleted EnrollmentStatus = "Completed" // candidate submitted the exam
	StatusRevoked   EnrollmentStatus = "Revoked"   // admin revoked access
)

// ExamEnrollment represents a single candidate's enrollment in an exam.
// access_token_hash and invitation_code_hash are stored as SHA-256 hex; neither
// raw value is ever persisted or returned in API responses.
type ExamEnrollment struct {
	ID                 uuid.UUID        `db:"id"                   json:"id"`
	EnterpriseID       uuid.UUID        `db:"enterprise_id"        json:"enterpriseId"`
	ExamID             uuid.UUID        `db:"exam_id"              json:"examId"`
	CandidateID        uuid.UUID        `db:"candidate_id"         json:"candidateId"`
	AccessTokenHash    string           `db:"access_token_hash"    json:"-"` // never serialised
	InvitationCodeHash *string          `db:"invitation_code_hash" json:"-"` // never serialised
	TokenExpiresAt     time.Time        `db:"token_expires_at"     json:"tokenExpiresAt"`
	MaxAttempts        int              `db:"max_attempts"         json:"maxAttempts"`
	AttemptsUsed       int              `db:"attempts_used"        json:"attemptsUsed"`
	Status             EnrollmentStatus `db:"status"               json:"status"`
	InvitationSentAt   *time.Time       `db:"invitation_sent_at"   json:"invitationSentAt,omitempty"`
	CreatedAt          time.Time        `db:"created_at"           json:"createdAt"`
}

// EnrollmentRepository is the persistence contract for exam enrollments.
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *ExamEnrollment) error
	CreateBulk(ctx context.Context, enrollments []*ExamEnrollment) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamEnrollment, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID, enterpriseID uuid.UUID) ([]*ExamEnrollment, error)
	GetByExamAndCandidate(ctx context.Context, examID uuid.UUID, candidateID uuid.UUID) (*ExamEnrollment, error)
	GetByInvitationCodeHash(ctx context.Context, codeHash string) (*ExamEnrollment, error)
	ListByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamEnrollment, int64, error)
	Update(ctx context.Context, enrollment *ExamEnrollment) error
	UpdateInvitation(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID, codeHash string, invitedAt time.Time) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status EnrollmentStatus) error
	IncrementAttempt(ctx context.Context, id uuid.UUID) error
	GetExpiredPendingEnrollments(ctx context.Context, limit int) ([]*ExamEnrollment, error)
	WithTx(tx pgx.Tx) EnrollmentRepository
}

// EnrollmentUseCase is the business-logic contract for the enrollment workflow.
type EnrollmentUseCase interface {
	// Phase 1 — Admin enrolls candidates. Generates tokens + opaque codes; stores
	// hashes; does NOT send any emails. Returns invitation URLs for each candidate.
	EnrollCandidates(ctx context.Context, enterpriseID uuid.UUID, examID uuid.UUID, candidateIDs []uuid.UUID, maxAttempts int, expiresAt time.Time) ([]*EnrollmentResult, error)

	// Phase 2 — Admin explicitly triggers email notification.
	NotifyCandidates(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, enrollmentIDs []uuid.UUID) ([]*NotifyResult, error)
	NotifyCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*NotifyResult, error)

	// Phase 3 — Candidate redeems opaque code from URL; receives JWT in response body.
	RedeemInvitationCode(ctx context.Context, code string) (string, error)

	// Phase 4 — Admin retrieves a fresh invitation URL (e.g. for no-email candidates).
	GetInvitationLink(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (string, error)

	// Standard read / management operations.
	GetEnrollmentsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamEnrollment, int64, error)
	GetEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamEnrollment, error)
	RevokeEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	ResetAttempts(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}
