package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
)

type enrollmentUseCase struct {
	pool   *pgxpool.Pool
	repo   domain.EnrollmentRepository
	logger *zap.Logger
}

func NewEnrollmentUseCase(pool *pgxpool.Pool, repo domain.EnrollmentRepository, logger *zap.Logger) domain.EnrollmentUseCase {
	return &enrollmentUseCase{
		pool:   pool,
		repo:   repo,
		logger: logger,
	}
}

// Generate a secure token using crypto/rand with a configurable length.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Hash a token with SHA-256
func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (uc *enrollmentUseCase) EnrollCandidates(ctx context.Context, enterpriseID uuid.UUID, examID uuid.UUID, candidateIDs []uuid.UUID, maxAttempts int, expiresAt time.Time) ([]string, error) {
	var rawTokens []string

	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		for _, cid := range candidateIDs {
			rawToken, err := generateSecureToken(16)
			if err != nil {
				return fmt.Errorf("generate secure token: %w", err)
			}

			hashedToken := hashToken(rawToken)

			enrollment := &domain.ExamEnrollment{
				EnterpriseID:    enterpriseID,
				ExamID:          examID,
				CandidateID:     cid,
				AccessTokenHash: hashedToken,
				TokenExpiresAt:  expiresAt,
				MaxAttempts:     maxAttempts,
				AttemptsUsed:    0,
			}

			if err := uc.repo.WithTx(tx).Create(ctx, enrollment); err != nil {
				return fmt.Errorf("create enrollment for candidate %s: %w", cid, err)
			}

			// Store raw token exactly once to hand back to the inviter
			rawTokens = append(rawTokens, rawToken)
		}
		return nil
	})

	if err != nil {
		uc.logger.Error("bulk enrollment failed", zap.Error(err), zap.String("examID", examID.String()))
		return nil, err
	}

	uc.logger.Info("candidates enrolled", zap.Int("count", len(candidateIDs)), zap.String("examID", examID.String()))
	return rawTokens, nil
}

func (uc *enrollmentUseCase) GetEnrollmentsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamEnrollment, int64, error) {
	return uc.repo.ListByExam(ctx, examID, enterpriseID, params)
}

func (uc *enrollmentUseCase) GetEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamEnrollment, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *enrollmentUseCase) RegenerateToken(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (string, error) {
	var rawToken string
	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		e, err := uc.repo.WithTx(tx).GetByID(ctx, id, enterpriseID)
		if err != nil {
			return err
		}

		rt, err := generateSecureToken(16)
		if err != nil {
			return err
		}
		rawToken = rt

		e.AccessTokenHash = hashToken(rawToken)

		if err := uc.repo.WithTx(tx).Update(ctx, e); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		uc.logger.Error("failed to regenerate token", zap.Error(err), zap.String("enrollmentID", id.String()))
		return "", err
	}

	uc.logger.Info("enrollment token regenerated", zap.String("enrollmentID", id.String()))
	return rawToken, nil
}

func (uc *enrollmentUseCase) RevokeEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return err
	}
	e.TokenExpiresAt = time.Now().Add(-1 * time.Hour)
	if err := uc.repo.Update(ctx, e); err != nil {
		uc.logger.Error("failed to revoke enrollment", zap.Error(err), zap.String("enrollmentID", id.String()))
		return err
	}
	uc.logger.Info("enrollment revoked", zap.String("enrollmentID", id.String()))
	return nil
}

func (uc *enrollmentUseCase) ResetAttempts(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return err
	}
	e.AttemptsUsed = 0
	return uc.repo.Update(ctx, e)
}
