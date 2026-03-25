package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RoleExamCandidate is embedded in every enrollment JWT as the "role" claim,
// allowing the api-gateway to map it to the correct access controls.
const RoleExamCandidate = "ExamCandidate"

type EnrollmentClaims struct {
	EnrollmentID uuid.UUID `json:"eid"`
	CandidateID  uuid.UUID `json:"cid"`
	ExamID       uuid.UUID `json:"xid"`
	EnterpriseID uuid.UUID `json:"ent"`
	Role         string    `json:"role"` // always "ExamCandidate"
	ExpiresAt    time.Time `json:"exp_at"`
}

type EnrollmentTokenService interface {
	GenerateToken(ctx context.Context, claims EnrollmentClaims) (string, error)
	ParseToken(ctx context.Context, token string) (*EnrollmentClaims, error)
}
