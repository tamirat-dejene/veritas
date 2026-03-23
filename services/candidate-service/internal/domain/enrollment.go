package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type ExamEnrollment struct {
	ID               uuid.UUID `db:"id" json:"id"`
	EnterpriseID     uuid.UUID `db:"enterprise_id" json:"enterpriseId"`
	ExamID           uuid.UUID `db:"exam_id" json:"examId"`
	CandidateID      uuid.UUID `db:"candidate_id" json:"candidateId"`
	AccessTokenHash  string    `db:"access_token_hash" json:"-"` // Never send in JSON
	TokenExpiresAt   time.Time `db:"token_expires_at" json:"tokenExpiresAt"`
	MaxAttempts      int       `db:"max_attempts" json:"maxAttempts"`
	AttemptsUsed     int       `db:"attempts_used" json:"attemptsUsed"`
	CreatedAt        time.Time `db:"created_at" json:"createdAt"`
}

type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *ExamEnrollment) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamEnrollment, error)
	GetByExamAndCandidate(ctx context.Context, examID uuid.UUID, candidateID uuid.UUID) (*ExamEnrollment, error)
	ListByExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamEnrollment, int64, error)
	Update(ctx context.Context, enrollment *ExamEnrollment) error
	IncrementAttempt(ctx context.Context, id uuid.UUID) error
	WithTx(tx pgx.Tx) EnrollmentRepository
}

type EnrollmentUseCase interface {
	EnrollCandidates(ctx context.Context, enterpriseID uuid.UUID, examID uuid.UUID, candidateIDs []uuid.UUID, maxAttempts int, expiresAt time.Time) ([]string, error) // Returns raw tokens mapped implicitly or wrapped
	GetEnrollmentsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*ExamEnrollment, int64, error)
	GetEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*ExamEnrollment, error)
	RegenerateToken(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (string, error) // Returns raw new token
	RevokeEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
	ResetAttempts(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}
