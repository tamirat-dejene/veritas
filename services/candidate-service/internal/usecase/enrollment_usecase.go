package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type enrollmentUseCase struct {
	pool           *pgxpool.Pool
	repo           domain.EnrollmentRepository
	candidateRepo  domain.CandidateRepository
	tokenService   domain.EnrollmentTokenService
	examClient     client.ExamServiceClient
	publisher      messaging.Publisher
	portalBaseURL  string
}

func NewEnrollmentUseCase(
	pool *pgxpool.Pool,
	repo domain.EnrollmentRepository,
	candidateRepo domain.CandidateRepository,
	tokenService domain.EnrollmentTokenService,
	examClient client.ExamServiceClient,
	publisher messaging.Publisher,
	portalBaseURL string,
) domain.EnrollmentUseCase {
	return &enrollmentUseCase{
		pool:          pool,
		repo:          repo,
		candidateRepo: candidateRepo,
		tokenService:  tokenService,
		examClient:    examClient,
		publisher:     publisher,
		portalBaseURL: portalBaseURL,
	}
}

// hashSHA256 returns the hex-encoded SHA-256 digest of s.
func hashSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// HashToken is kept as an exported alias used by the session usecase.
func HashToken(token string) string { return hashSHA256(token) }

// generateOpaqueCode returns a cryptographically random 32-byte hex string
// suitable for use as a single-use invitation code in a URL.
func generateOpaqueCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate opaque code: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// buildInvitationURL constructs the candidate-facing URL that contains only
// the opaque code — the raw JWT never appears in a URL.
func (uc *enrollmentUseCase) buildInvitationURL(opaqueCode string) string {
	return fmt.Sprintf("%s/exam/start?code=%s", uc.portalBaseURL, opaqueCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 1: EnrollCandidates
// Creates DB records + generates JWT hash + opaque code hash.
// Does NOT send any emails. Returns invitation URLs for admin use.
// ─────────────────────────────────────────────────────────────────────────────

func (uc *enrollmentUseCase) EnrollCandidates(
	ctx context.Context,
	enterpriseID uuid.UUID,
	examID uuid.UUID,
	candidateIDs []uuid.UUID,
	maxAttempts int,
	expiresAt time.Time,
) ([]*domain.EnrollmentResult, error) {
	results := make([]*domain.EnrollmentResult, 0, len(candidateIDs))

	err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		for _, cid := range candidateIDs {
			enrollmentID := uuid.New()

			// Generate the JWT that the candidate will use to authenticate.
			claims := domain.EnrollmentClaims{
				EnrollmentID: enrollmentID,
				CandidateID:  cid,
				ExamID:       examID,
				EnterpriseID: enterpriseID,
				Role:         domain.RoleExamCandidate,
				ExpiresAt:    expiresAt,
			}
			rawToken, err := uc.tokenService.GenerateToken(ctx, claims)
			if err != nil {
				return fmt.Errorf("generate enrollment token for candidate %s: %w", cid, err)
			}

			// Generate the opaque invitation code (goes in the URL, never the JWT).
			opaqueCode, err := generateOpaqueCode()
			if err != nil {
				return fmt.Errorf("generate opaque code for candidate %s: %w", cid, err)
			}

			enrollment := &domain.ExamEnrollment{
				ID:                 enrollmentID,
				EnterpriseID:       enterpriseID,
				ExamID:             examID,
				CandidateID:        cid,
				AccessTokenHash:    hashSHA256(rawToken),
				InvitationCodeHash: hashSHA256(opaqueCode),
				TokenExpiresAt:     expiresAt,
				MaxAttempts:        maxAttempts,
				AttemptsUsed:       0,
				Status:             domain.StatusPending,
			}

			if err := uc.repo.WithTx(tx).Create(ctx, enrollment); err != nil {
				return fmt.Errorf("create enrollment for candidate %s: %w", cid, err)
			}

			results = append(results, &domain.EnrollmentResult{
				EnrollmentID:  enrollmentID,
				CandidateID:   cid,
				InvitationURL: uc.buildInvitationURL(opaqueCode),
				Status:        domain.StatusPending,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 2: NotifyCandidates / NotifyCandidate
// Admin-triggered. Rotates the opaque code (invalidates old one), publishes
// Kafka event, updates status to Invited.
// ─────────────────────────────────────────────────────────────────────────────

func (uc *enrollmentUseCase) NotifyCandidates(
	ctx context.Context,
	examID uuid.UUID,
	enterpriseID uuid.UUID,
	enrollmentIDs []uuid.UUID,
) ([]*domain.NotifyResult, error) {
	results := make([]*domain.NotifyResult, 0, len(enrollmentIDs))
	for _, eid := range enrollmentIDs {
		r, err := uc.NotifyCandidate(ctx, eid, enterpriseID)
		if err != nil {
			return nil, fmt.Errorf("notify enrollment %s: %w", eid, err)
		}
		results = append(results, r)
	}
	return results, nil
}

func (uc *enrollmentUseCase) NotifyCandidate(
	ctx context.Context,
	id uuid.UUID,
	enterpriseID uuid.UUID,
) (*domain.NotifyResult, error) {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return nil, err
	}
	if e.Status == domain.StatusRevoked {
		return nil, domain.ErrInvalidAccessToken
	}

	// Fetch candidate profile for email and name.
	candidate, err := uc.candidateRepo.GetByID(ctx, e.CandidateID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("fetch candidate for notification: %w", err)
	}

	if candidate.Email == nil {
		return &domain.NotifyResult{
			EnrollmentID: e.ID,
			CandidateID:  e.CandidateID,
			NotifyStatus: "skipped_no_email",
		}, nil
	}

	// Fetch exam metadata for the exam title.
	exam, err := uc.examClient.GetExamMetadata(ctx, e.ExamID)
	if err != nil {
		return nil, fmt.Errorf("fetch exam metadata: %w", err)
	}

	// Rotate the opaque code on every notify to invalidate any previously
	// copied but undelivered URLs.
	opaqueCode, err := generateOpaqueCode()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	e.InvitationCodeHash = hashSHA256(opaqueCode)
	e.Status = domain.StatusInvited
	e.InvitationSentAt = &now

	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("update enrollment on notify: %w", err)
	}

	invitationURL := uc.buildInvitationURL(opaqueCode)
	candidateName := candidate.FirstName + " " + candidate.LastName

	event := domain.CandidateEnrollmentInvitedEvent{
		EnrollmentID:   e.ID,
		CandidateID:    e.CandidateID,
		ExamID:         e.ExamID,
		EnterpriseID:   e.EnterpriseID,
		CandidateName:  candidateName,
		CandidateEmail: *candidate.Email,
		ExamTitle:      exam.Title,
		InvitationURL:  invitationURL,
		ExpiresAt:      e.TokenExpiresAt,
		Timestamp:      now.UnixMilli(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal invitation event: %w", err)
	}

	if err := uc.publisher.Publish(ctx, messaging.Message{
		Topic: topics.CandidateEnrollmentInvited,
		Key:   []byte(e.ID.String()),
		Value: payload,
	}); err != nil {
		return nil, fmt.Errorf("publish invitation event: %w", err)
	}

	return &domain.NotifyResult{
		EnrollmentID: e.ID,
		CandidateID:  e.CandidateID,
		NotifyStatus: "sent",
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 3: RedeemInvitationCode
// Candidate exchanges the opaque URL code for a raw JWT (returned in body only).
// ─────────────────────────────────────────────────────────────────────────────

func (uc *enrollmentUseCase) RedeemInvitationCode(ctx context.Context, code string) (string, error) {
	codeHash := hashSHA256(code)
	e, err := uc.repo.GetByInvitationCodeHash(ctx, codeHash)
	if err != nil {
		return "", domain.ErrInvalidAccessToken
	}

	if e.Status == domain.StatusRevoked {
		return "", domain.ErrInvalidAccessToken
	}
	if time.Now().After(e.TokenExpiresAt) {
		return "", domain.ErrInvalidAccessToken
	}
	if e.AttemptsUsed >= e.MaxAttempts {
		return "", domain.ErrMaxAttemptsReached
	}

	// Regenerate the JWT from stored enrollment claims using the same secret.
	// The raw token is never persisted, but the claims are — so we can recreate it.
	claims := domain.EnrollmentClaims{
		EnrollmentID: e.ID,
		CandidateID:  e.CandidateID,
		ExamID:       e.ExamID,
		EnterpriseID: e.EnterpriseID,
		Role:         domain.RoleExamCandidate,
		ExpiresAt:    e.TokenExpiresAt,
	}
	rawToken, err := uc.tokenService.GenerateToken(ctx, claims)
	if err != nil {
		return "", fmt.Errorf("regenerate token on redeem: %w", err)
	}

	// Verify the regenerated token hash matches what is stored (integrity check).
	if hashSHA256(rawToken) != e.AccessTokenHash {
		return "", domain.ErrInvalidAccessToken
	}

	// Update status to Opened (idempotent — already Opened is fine).
	if e.Status == domain.StatusPending || e.Status == domain.StatusInvited {
		_ = uc.repo.UpdateStatus(ctx, e.ID, domain.StatusOpened)
	}

	return rawToken, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase 4: GetInvitationLink
// Admin retrieves a fresh invitation URL for manual distribution.
// Rotates the opaque code so any previously shared URL is invalidated.
// ─────────────────────────────────────────────────────────────────────────────

func (uc *enrollmentUseCase) GetInvitationLink(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (string, error) {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return "", err
	}
	if e.Status == domain.StatusRevoked {
		return "", domain.ErrInvalidAccessToken
	}

	opaqueCode, err := generateOpaqueCode()
	if err != nil {
		return "", err
	}
	e.InvitationCodeHash = hashSHA256(opaqueCode)
	if err := uc.repo.Update(ctx, e); err != nil {
		return "", err
	}
	return uc.buildInvitationURL(opaqueCode), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Standard read / management operations (unchanged semantics)
// ─────────────────────────────────────────────────────────────────────────────

func (uc *enrollmentUseCase) GetEnrollmentsForExam(ctx context.Context, examID uuid.UUID, enterpriseID uuid.UUID, params pagination.Params) ([]*domain.ExamEnrollment, int64, error) {
	return uc.repo.ListByExam(ctx, examID, enterpriseID, params)
}

func (uc *enrollmentUseCase) GetEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) (*domain.ExamEnrollment, error) {
	return uc.repo.GetByID(ctx, id, enterpriseID)
}

func (uc *enrollmentUseCase) RevokeEnrollment(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return err
	}
	e.Status = domain.StatusRevoked
	e.TokenExpiresAt = time.Now().Add(-1 * time.Hour) // belt-and-suspenders expiry
	return uc.repo.Update(ctx, e)
}

func (uc *enrollmentUseCase) ResetAttempts(ctx context.Context, id uuid.UUID, enterpriseID uuid.UUID) error {
	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return err
	}
	e.AttemptsUsed = 0
	return uc.repo.Update(ctx, e)
}
