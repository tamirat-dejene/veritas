package domain

import (
	"context"

	"github.com/google/uuid"
)

type EnrollmentClaims struct {
	EnrollmentID uuid.UUID `json:"eid"`
	CandidateID  uuid.UUID `json:"cid"`
	ExamID       uuid.UUID `json:"xid"`
	EnterpriseID uuid.UUID `json:"ent"`
}

type EnrollmentTokenService interface {
	GenerateToken(ctx context.Context, claims EnrollmentClaims) (string, error)
	ParseToken(ctx context.Context, token string) (*EnrollmentClaims, error)
}
