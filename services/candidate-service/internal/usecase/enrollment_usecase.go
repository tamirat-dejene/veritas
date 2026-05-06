package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/infrastructure/client"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging"
	"github.com/tamirat-dejene/veritas/shared/pkg/messaging/topics"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"golang.org/x/sync/errgroup"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
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
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
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
	// Validate exam status: must be Scheduled to allow enrollment
	exam, err := uc.examClient.GetExamMetadata(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch exam metadata for enrollment: %w", err)
	}
	if exam.Status != sdomain.ExamScheduled {
		return nil, domain.ErrInvalidExamStatus
	}

	// Validate expiration time
	if expiresAt.Before(time.Now()) || expiresAt.After(*exam.ScheduledEnd) {
		return nil, domain.ErrInvalidEnrollmentTime
	}

	const batchSize = 100
	results := make([]*domain.EnrollmentResult, 0, len(candidateIDs))

	for i := 0; i < len(candidateIDs); i += batchSize {
		end := min(i + batchSize, len(candidateIDs))

		batchIDs := candidateIDs[i:end]
		batchEnrollments := make([]*domain.ExamEnrollment, 0, len(batchIDs))
		batchResults := make([]*domain.EnrollmentResult, 0, len(batchIDs))

		err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
			for _, cid := range batchIDs {
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

					h := hashSHA256(opaqueCode)
					enrollment := &domain.ExamEnrollment{
						ID:                 enrollmentID,
						EnterpriseID:       enterpriseID,
						ExamID:             examID,
						CandidateID:        cid,
						AccessTokenHash:    hashSHA256(rawToken),
						InvitationCodeHash: &h,
						TokenExpiresAt:     expiresAt,
					MaxAttempts:        maxAttempts,
					AttemptsUsed:       0,
					Status:             domain.StatusPending,
					CreatedAt:          time.Now(),
				}

				batchEnrollments = append(batchEnrollments, enrollment)
				batchResults = append(batchResults, &domain.EnrollmentResult{
					EnrollmentID:  enrollmentID,
					CandidateID:   cid,
					InvitationURL: uc.buildInvitationURL(opaqueCode),
					Status:        domain.StatusPending,
				})
			}

			if err := uc.repo.WithTx(tx).CreateBulk(ctx, batchEnrollments); err != nil {
				return fmt.Errorf("bulk create enrollments: %w", err)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
		results = append(results, batchResults...)
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
	if len(enrollmentIDs) == 0 {
		return nil, nil
	}

	// 1. Fetch data upfront
	exam, err := uc.examClient.GetExamMetadata(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("fetch exam metadata: %w", err)
	}

	enrollments, err := uc.repo.GetByIDs(ctx, enrollmentIDs, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("fetch enrollments: %w", err)
	}

	candidateIDs := make([]uuid.UUID, 0, len(enrollments))
	enrollmentMap := make(map[uuid.UUID]*domain.ExamEnrollment)
	for _, e := range enrollments {
		candidateIDs = append(candidateIDs, e.CandidateID)
		enrollmentMap[e.ID] = e
	}

	candidates, err := uc.candidateRepo.GetByIDs(ctx, candidateIDs, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("fetch candidates: %w", err)
	}

	candidateMap := make(map[uuid.UUID]*domain.CandidateProfile)
	for _, c := range candidates {
		candidateMap[c.ID] = c
	}

	// 2. Process with bounded concurrency
	results := make([]*domain.NotifyResult, len(enrollmentIDs))
	events := make([]messaging.Message, 0, len(enrollmentIDs))
	var eventMu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5) // Bounded concurrency

	for i, eid := range enrollmentIDs {
		g.Go(func() error {
			e, ok := enrollmentMap[eid]
			if !ok {
				results[i] = &domain.NotifyResult{
					EnrollmentID: eid,
					NotifyStatus: "failed_not_found",
				}
				return nil
			}

			if e.Status == domain.StatusRevoked {
				results[i] = &domain.NotifyResult{
					EnrollmentID: eid,
					NotifyStatus: "failed_revoked",
				}
				return nil
			}

			candidate, ok := candidateMap[e.CandidateID]
			if !ok || candidate.Email == nil {
				results[i] = &domain.NotifyResult{
					EnrollmentID: eid,
					CandidateID:  e.CandidateID,
					NotifyStatus: "skipped_no_email",
				}
				return nil
			}

			opaqueCode, err := generateOpaqueCode()
			if err != nil {
				return err
			}

			now := time.Now()
			codeHash := hashSHA256(opaqueCode)

			// Partial update: only invitation hash and status
			if err := uc.repo.UpdateInvitation(ctx, e.ID, enterpriseID, codeHash, now); err != nil {
				return fmt.Errorf("update enrollment %s: %w", e.ID, err)
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
				return fmt.Errorf("marshal event for %s: %w", e.ID, err)
			}

			eventMu.Lock()
			events = append(events, messaging.Message{
				Topic: topics.CandidateEnrollmentInvited,
				Key:   []byte(e.ID.String()),
				Value: payload,
			})
			eventMu.Unlock()

			results[i] = &domain.NotifyResult{
				EnrollmentID: e.ID,
				CandidateID:  e.CandidateID,
				NotifyStatus: "sent",
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 3. Batch publish Kafka events
	if len(events) > 0 {
		if err := uc.publisher.PublishBatch(ctx, events); err != nil {
			return nil, fmt.Errorf("publish batch events: %w", err)
		}
	}

	return results, nil
}

func (uc *enrollmentUseCase) NotifyCandidate(
	ctx context.Context,
	id uuid.UUID,
	enterpriseID uuid.UUID,
) (*domain.NotifyResult, error) {
	// Re-use batch logic for single notification if possible, or keep simple
	results, err := uc.NotifyCandidates(ctx, uuid.Nil, enterpriseID, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, domain.ErrEnrollmentNotFound
	}

	e, err := uc.repo.GetByID(ctx, id, enterpriseID)
	if err != nil {
		return nil, err
	}
	
	results, err = uc.NotifyCandidates(ctx, e.ExamID, enterpriseID, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	return results[0], nil
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
	h := hashSHA256(opaqueCode)
	e.InvitationCodeHash = &h
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
