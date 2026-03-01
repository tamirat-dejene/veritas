package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/auth-service/internal/usecase"
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
	Email    string `json:"email"    binding:"required,email"    example:"admin@veritas.io"`
	Password string `json:"password" binding:"required,min=1"    example:"s3cur3P@ssw0rd"`
}

// refreshRequest is the validated JSON body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// logoutRequest is the validated JSON body for POST /auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// Login handles POST /auth/login.
//
//	@Summary		Authenticate a user
//	@Description	Validates email/password credentials and returns a JWT access token plus a refresh token.
//	@Description	Only users with roles SystemAdmin, EnterpriseAdmin, EnterpriseAuto, or EnterpriseStaff may authenticate via this service.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login credentials"
//	@Success		200		{object}	TokenResponse	"JWT access and refresh tokens"
//	@Failure		400		{object}	ErrorResponse	"Missing or malformed request body"
//	@Failure		401		{object}	ErrorResponse	"Invalid email or password / expired/revoked token"
//	@Failure		403		{object}	ErrorResponse	"Account locked, inactive, deleted, or role not permitted"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
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
//	@Description	Exchanges a valid refresh token for a new JWT access token and a rotated refresh token. The old refresh token is invalidated.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		refreshRequest	true	"Refresh token"
//	@Success		200		{object}	TokenResponse	"New JWT access and refresh tokens"
//	@Failure		400		{object}	ErrorResponse	"Missing or malformed request body"
//	@Failure		401		{object}	ErrorResponse	"Refresh token invalid, revoked, or expired"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
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
//	@Description	Immediately revokes the supplied refresh token so it cannot be used for future token refreshes.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	logoutRequest	true	"Refresh token to revoke"
//	@Success		204		"Token revoked — no content"
//	@Failure		400		{object}	ErrorResponse	"Missing or malformed request body"
//	@Failure		401		{object}	ErrorResponse	"Refresh token invalid, revoked, or expired"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
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

// handleUseCaseError maps domain errors to HTTP status codes.
func (h *AuthHandler) handleUseCaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUserNotFound):
		writeError(c, http.StatusUnauthorized, "invalid email or password")

	case errors.Is(err, domain.ErrAccountLocked):
		writeError(c, http.StatusForbidden, "account is temporarily locked")

	case errors.Is(err, domain.ErrUserInactive),
		errors.Is(err, domain.ErrUserDeleted),
		errors.Is(err, domain.ErrRoleNotPermitted):
		writeError(c, http.StatusForbidden, "access denied")

	case errors.Is(err, domain.ErrTokenNotFound),
		errors.Is(err, domain.ErrTokenRevoked),
		errors.Is(err, domain.ErrTokenExpired):
		writeError(c, http.StatusUnauthorized, "invalid or expired token")

	default:
		h.log.Error("unhandled use case error", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
