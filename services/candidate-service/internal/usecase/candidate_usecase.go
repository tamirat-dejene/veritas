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
	"go.uber.org/zap"
)

type candidateUseCase struct {
	pool   *pgxpool.Pool
	repo   domain.CandidateRepository
	logger *zap.Logger
}

func NewCandidateUseCase(pool *pgxpool.Pool, repo domain.CandidateRepository, logger *zap.Logger) domain.CandidateUseCase {
	return &candidateUseCase{
		pool:   pool,
		repo:   repo,
		logger: logger,
	}
}

func (uc *candidateUseCase) CreateCandidate(ctx context.Context, candidate *domain.CandidateProfile) (*domain.CandidateProfile, error) {
	if err := uc.repo.Create(ctx, candidate); err != nil {
		uc.logger.Error("failed to create candidate", zap.Error(err), zap.String("enterpriseID", candidate.EnterpriseID.String()), zap.String("externalID", candidate.ExternalID))
		return nil, err
	}
	uc.logger.Info("candidate created", zap.String("candidateID", candidate.ID.String()), zap.String("enterpriseID", candidate.EnterpriseID.String()))
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
	for i, row := range records {
		if len(row) < 3 {
			uc.logger.Warn("skipping invalid row", zap.Int("row", i+2))
			continue
		}

		c := &domain.CandidateProfile{
			EnterpriseID: enterpriseID,
			ExternalID:   row[0],
			FirstName:    row[1],
			LastName:     row[2],
			IsActive:     true,
		}

		if len(row) >= 4 {
			email := row[3]
			if email != "" {
				c.Email = &email
			}
		}

		candidates = append(candidates, c)
	}

	if len(candidates) == 0 {
		return 0, fmt.Errorf("no valid candidates found in CSV")
	}

	err = RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		return uc.repo.WithTx(tx).CreateBulk(ctx, candidates)
	})
	if err != nil {
		uc.logger.Error("bulk upload failed", zap.Error(err), zap.String("enterpriseID", enterpriseID.String()))
		return 0, err
	}

	uc.logger.Info("bulk upload successful", zap.Int("count", len(candidates)), zap.String("enterpriseID", enterpriseID.String()))
	return len(candidates), nil
}

func (uc *candidateUseCase) GetCandidates(ctx context.Context, enterpriseID uuid.UUID) ([]*domain.CandidateProfile, error) {
	return uc.repo.ListByEnterprise(ctx, enterpriseID)
}

func (uc *candidateUseCase) GetCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.CandidateProfile, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *candidateUseCase) UpdateCandidate(ctx context.Context, candidate *domain.CandidateProfile) error {
	return uc.repo.Update(ctx, candidate)
}

func (uc *candidateUseCase) DeactivateCandidate(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	if err := uc.repo.Deactivate(ctx, id, enterpriseID); err != nil {
		uc.logger.Error("failed to deactivate candidate", zap.Error(err), zap.String("candidateID", id.String()))
		return err
	}
	uc.logger.Info("candidate deactivated", zap.String("candidateID", id.String()), zap.String("enterpriseID", enterpriseID.String()))
	return nil
}
