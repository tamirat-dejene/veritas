package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// LoginInput is the request payload for the login use case.
type LoginInput struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

// LoginOutput is the response payload for the login use case.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// LoginUseCase orchestrates user authentication with security auditing.
type LoginUseCase struct {
	pool                    *pgxpool.Pool
	enterpriseServiceClient domain.EnterpriseServiceClient
	refreshTokenRepo        domain.RefreshTokenRepository
	jwtService              domain.TokenService
	refreshService          domain.TokenService
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
	eventPublisher          domain.EventPublisher
	log                     *zap.Logger
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(
	pool *pgxpool.Pool,
	enterpriseServiceClient domain.EnterpriseServiceClient,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService domain.TokenService,
	refreshService domain.TokenService,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	eventPublisher domain.EventPublisher,
	log *zap.Logger,
) *LoginUseCase {
	return &LoginUseCase{
		pool:                    pool,
		enterpriseServiceClient: enterpriseServiceClient,
		refreshTokenRepo:        refreshTokenRepo,
		jwtService:              jwtService,
		refreshService:          refreshService,
		accessTokenTTL:          accessTokenTTL,
		refreshTokenTTL:         refreshTokenTTL,
		eventPublisher:          eventPublisher,
		log:                     log,
	}
}

// Execute authenticates the user and returns a token pair on success.
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// 1. Find user by email.
	l := logger.WithContext(ctx, uc.log)
	user, err := uc.enterpriseServiceClient.FindByEmail(ctx, input.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			l.Warn("login attempt for unknown email", zap.String("email", input.Email))
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("LoginUseCase.Execute: FindByEmail: %w", err)
	}

	// 2. Reject if soft-deleted or inactive.
	if user.IsDeleted {
		l.Warn("login attempt for deleted user", zap.String("userId", user.ID.String()))
		return nil, domain.ErrUserDeleted
	}
	if !user.IsActive {
		l.Warn("login attempt for inactive user", zap.String("userId", user.ID.String()))
		return nil, domain.ErrUserInactive
	}

	// 3. Check Account Lock.
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		l.Warn("login attempt for locked account", zap.String("userId", user.ID.String()), zap.Time("lockedUntil", *user.LockedUntil))
		return nil, domain.ErrAccountLocked
	}

	// 4. Reject roles not served by this service.
	if _, ok := domain.AllowedAuthRoles[user.Role]; !ok {
		l.Warn("login attempt for unauthorized role", zap.String("userId", user.ID.String()), zap.String("role", string(user.Role)))
		return nil, domain.ErrRoleNotPermitted
	}

	// 5. Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		l.Warn("invalid password attempt", zap.String("userId", user.ID.String()))

		var lockUntil *time.Time
		if user.FailedLoginAttempts+1 >= 5 {
			lockTime := time.Now().Add(15 * time.Minute)
			lockUntil = &lockTime
			l.Info("account locked due to too many failures", zap.String("userId", user.ID.String()))
		}

		user.FailedLoginAttempts += 1
		go func() {
			if errUpdate := uc.enterpriseServiceClient.UpdateLoginFailure(context.Background(), user.ID, lockUntil, user.FailedLoginAttempts); errUpdate != nil {
				l.Error("failed to update login failure stats", zap.Error(errUpdate))
			}
		}()

		return nil, domain.ErrInvalidCredentials
	}

	// 6. Generate access token (JWT).
	accessToken, err := uc.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateAccessToken: %w", err)
	}

	// 7. Generate refresh token hash.
	rawRefreshToken, tokenHash, err := uc.refreshService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("LoginUseCase.Execute: GenerateRefreshToken: %w", err)
	}

	// 8. Persist refresh token hash.
	if err := RunInTx(ctx, uc.pool, func(tx pgx.Tx) error {
		rt := domain.NewRefreshToken(user.ID, tokenHash, uc.refreshTokenTTL)
		if err := uc.refreshTokenRepo.WithTx(tx).Create(ctx, rt); err != nil {
			return fmt.Errorf("create refresh token: %w", err)
		}
		return nil
	}); err != nil {
		l.Error("login transaction failed", zap.Error(err))
		return nil, fmt.Errorf("LoginUseCase.Execute transaction: %w", err)
	}

	l.Info("user logged in successfully", zap.String("userId", user.ID.String()))

	// 9. Update Login Audit Stats & Publish Login Event
	go func() {
		if err := uc.enterpriseServiceClient.UpdateLoginSuccess(context.Background(), user.ID, input.IP, input.UserAgent); err != nil {
			l.Error("failed to update login success stats", zap.Error(err))
		}

		if err := uc.eventPublisher.PublishLogin(context.Background(), user.ID, user.Email); err != nil {
			l.Error("failed to publish login event", zap.Error(err))
		}
	}()

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(uc.accessTokenTTL.Seconds()),
	}, nil
}
