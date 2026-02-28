package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CandidateProfile struct {
	ID               uuid.UUID `db:"id" json:"id"`
	EnterpriseID     uuid.UUID `db:"enterprise_id" json:"enterpriseId"`
	ExternalID       string    `db:"external_id" json:"externalId"`
	FirstName        string    `db:"first_name" json:"firstName"`
	LastName         string    `db:"last_name" json:"lastName"`
	Email            *string   `db:"email" json:"email,omitempty"`
	FaceReferenceURL *string   `db:"face_reference_url" json:"faceReferenceUrl,omitempty"`
	IsActive         bool      `db:"is_active" json:"isActive"`
	CreatedAt        time.Time `db:"created_at" json:"createdAt"`
}

type CandidateRepository interface {
	Create(ctx context.Context, candidate *CandidateProfile) error
	CreateBulk(ctx context.Context, candidates []*CandidateProfile) error
	GetByID(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*CandidateProfile, error)
	GetByExternalID(ctx context.Context, externalID string, enterpriseID uuid.UUID) (*CandidateProfile, error)
	ListByEnterprise(ctx context.Context, enterpriseID uuid.UUID) ([]*CandidateProfile, error)
	Update(ctx context.Context, candidate *CandidateProfile) error
	Deactivate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}

type CandidateUseCase interface {
	CreateCandidate(ctx context.Context, candidate *CandidateProfile) (*CandidateProfile, error)
	BulkUpload(ctx context.Context, enterpriseID uuid.UUID, csvData []byte) (int, error) // Returns number of created records
	GetCandidates(ctx context.Context, enterpriseID uuid.UUID) ([]*CandidateProfile, error)
	GetCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*CandidateProfile, error)
	UpdateCandidate(ctx context.Context, candidate *CandidateProfile) error
	DeactivateCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error
}
