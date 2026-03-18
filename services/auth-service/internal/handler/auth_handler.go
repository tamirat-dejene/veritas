package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/usecase"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

// AuthHandler holds references to the three auth use cases.
type AuthHandler struct {
	loginUseCase   *usecase.LoginUseCase
	refreshUseCase *usecase.RefreshUseCase
	logoutUseCase  *usecase.LogoutUseCase
	log            *zap.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	loginUC *usecase.LoginUseCase,
	refreshUC *usecase.RefreshUseCase,
	logoutUC *usecase.LogoutUseCase,
	log *zap.Logger,
) *AuthHandler {
	return &AuthHandler{
		loginUseCase:   loginUC,
		refreshUseCase: refreshUC,
		logoutUseCase:  logoutUC,
		log:            log,
	}
}

// loginRequest is the validated JSON body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"    example:"admin@veritas.io" format:"email"`
	Password string `json:"password" binding:"required,min=1"    example:"s3cur3P@ssw0rd"`
}

// refreshRequest is the validated JSON body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,len=64,hexadecimal" example:"b8a54f0f0cc6d2f68dd0b457ea4bb7f814ff69ec487f474f5c6f1781b6f0a0d3" minLength:"64" maxLength:"64" pattern:"^[a-f0-9]{64}$"`
}

// logoutRequest is the validated JSON body for POST /auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,len=64,hexadecimal" example:"b8a54f0f0cc6d2f68dd0b457ea4bb7f814ff69ec487f474f5c6f1781b6f0a0d3" minLength:"64" maxLength:"64" pattern:"^[a-f0-9]{64}$"`
}

// Login handles POST /auth/login.
//
//	@Summary		Authenticate a user
//	@ID			authLogin
//	@Description	Validates email/password credentials and returns a JWT access token plus a refresh token.
//	@Description	Only users with roles SystemAdmin, EnterpriseAdmin, EnterpriseAuto, or EnterpriseStaff may authenticate via this service.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login credentials"
//	@Success		200		{object}	TokenResponse	"JWT access and refresh tokens"
//	@Header			200		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		400		{object}	BadRequestErrorResponse	"Missing or malformed request body"
//	@Header			400		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		401		{object}	UnauthorizedErrorResponse	"Invalid email or password"
//	@Header			401		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		403		{object}	ForbiddenErrorResponse	"Account locked, inactive, deleted, or role not permitted"
//	@Header			403		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		500		{object}	InternalErrorResponse	"Internal server error"
//	@Header			500		{string}	X-Request-ID	"Request correlation ID"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	out, err := h.loginUseCase.Execute(c.Request.Context(), usecase.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	writeTokens(c, out.AccessToken, out.RefreshToken, out.ExpiresIn)
}

// Refresh handles POST /auth/refresh.
//
//	@Summary		Refresh an access token
//	@ID			authRefresh
//	@Description	Exchanges a valid refresh token for a new JWT access token and a rotated refresh token. The old refresh token is invalidated.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		refreshRequest	true	"Refresh token"
//	@Success		200		{object}	TokenResponse	"New JWT access and refresh tokens"
//	@Header			200		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		400		{object}	BadRequestErrorResponse	"Missing or malformed request body"
//	@Header			400		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		401		{object}	UnauthorizedErrorResponse	"Refresh token invalid, revoked, or expired"
//	@Header			401		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		500		{object}	InternalErrorResponse	"Internal server error"
//	@Header			500		{string}	X-Request-ID	"Request correlation ID"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	out, err := h.refreshUseCase.Execute(c.Request.Context(), usecase.RefreshInput{
		RefreshToken: strings.TrimSpace(req.RefreshToken),
	})
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	writeTokens(c, out.AccessToken, out.RefreshToken, out.ExpiresIn)
}

// Logout handles POST /auth/logout.
//
//	@Summary		Revoke a refresh token
//	@ID			authLogout
//	@Description	Immediately revokes the supplied refresh token so it cannot be used for future token refreshes.
//	@Description	This endpoint is idempotent: invalid, unknown, expired, or already-revoked tokens return 204 to prevent token enumeration.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	logoutRequest	true	"Refresh token to revoke"
//	@Success		204		"Token revoked — no content"
//	@Header			204		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		400		{object}	BadRequestErrorResponse	"Missing or malformed request body"
//	@Header			400		{string}	X-Request-ID	"Request correlation ID"
//	@Failure		500		{object}	InternalErrorResponse	"Internal server error"
//	@Header			500		{string}	X-Request-ID	"Request correlation ID"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := h.logoutUseCase.Execute(c.Request.Context(), usecase.LogoutInput{
		RefreshToken: strings.TrimSpace(req.RefreshToken),
	}); err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// handleUseCaseError maps domain errors to HTTP status codes and logs warnings/errors.
func (h *AuthHandler) handleUseCaseError(c *gin.Context, err error) {
	l := logger.WithContext(c.Request.Context(), h.log)

	switch {
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUserNotFound):
		l.Warn("Authentication failed: invalid credentials", zap.String("ip", c.ClientIP()))
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")

	case errors.Is(err, domain.ErrAccountLocked):
		l.Warn("Authentication failed: account locked", zap.String("ip", c.ClientIP()))
		writeError(c, http.StatusForbidden, "account_locked", "account is temporarily locked")

	case errors.Is(err, domain.ErrUserInactive),
		errors.Is(err, domain.ErrUserDeleted),
		errors.Is(err, domain.ErrRoleNotPermitted):
		l.Warn("Authentication failed: access denied", zap.Error(err), zap.String("ip", c.ClientIP()))
		writeError(c, http.StatusForbidden, "access_denied", "access denied")

	case errors.Is(err, domain.ErrTokenNotFound),
		errors.Is(err, domain.ErrTokenRevoked),
		errors.Is(err, domain.ErrTokenExpired):
		l.Warn("Token rotation failed: invalid or expired token", zap.Error(err), zap.String("ip", c.ClientIP()))
		writeError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")

	default:
		l.Error("Unhandled use case error", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
