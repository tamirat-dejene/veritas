package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type candidateUseCase struct {
	pool *pgxpool.Pool
	repo domain.CandidateRepository
}

func NewCandidateUseCase(pool *pgxpool.Pool, repo domain.CandidateRepository) domain.CandidateUseCase {
	return &candidateUseCase{
		pool: pool,
		repo: repo,
	}
}

func (uc *candidateUseCase) CreateCandidate(ctx context.Context, candidate *domain.CandidateProfile) (*domain.CandidateProfile, error) {
	if err := uc.repo.Create(ctx, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (uc *candidateUseCase) BulkUpload(ctx context.Context, enterpriseID uuid.UUID, csvData []byte) (int, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))

	// Read header: usually external_id, first_name, last_name, email (optional)
	_, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV rows: %w", err)
	}

	var candidates []*domain.CandidateProfile
	for _, row := range records {
		if len(row) < 3 {
			continue
		}

		c := &domain.CandidateProfile{
			EnterpriseID: enterpriseID,
			ExternalID:   row[0],
			FirstName:    row[1],
			LastName:     row[2],
			IsActive:     true,
		}

		if len(row) >= 4 && row[3] != "" {
			email := row[3]
			c.Email = &email
		}

		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		return 0, domain.ErrNoValidCandidates
	}

	err = RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		return uc.repo.WithTx(tx).CreateBulk(ctx, candidates)
	})
	if err != nil {
		return 0, err
	}

	return len(candidates), nil
}

func (uc *candidateUseCase) GetCandidates(ctx context.Context, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.CandidateProfile, int64, error) {
	return uc.repo.ListByEnterprise(ctx, enterpriseID, params)
}

func (uc *candidateUseCase) GetCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.CandidateProfile, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *candidateUseCase) UpdateCandidate(ctx context.Context, candidate *domain.CandidateProfile) error {
	existing, err := uc.repo.GetByID(ctx, candidate.ID, candidate.EnterpriseID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrCandidateNotFound
	}

	existing.FirstName = candidate.FirstName
	existing.LastName = candidate.LastName
	if candidate.Email != nil {
		existing.Email = candidate.Email
	}

	return uc.repo.Update(ctx, existing)
}

func (uc *candidateUseCase) DeactivateCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	if err := uc.repo.Deactivate(ctx, id, enterpriseID); err != nil {
		return err
	}
	return nil
}

func (uc *candidateUseCase) GetEmailsByExamID(ctx context.Context, examID, enterpriseID uuid.UUID) ([]string, error) {
	return uc.repo.GetEmailsByExamID(ctx, examID, enterpriseID)
}
