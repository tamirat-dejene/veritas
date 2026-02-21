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
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

// refreshRequest is the validated JSON body for POST /auth/refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// logoutRequest is the validated JSON body for POST /auth/logout.
type logoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Trim whitespace to prevent trivial bypass attempts.
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	out, err := h.loginUseCase.Execute(c.Request.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	writeTokens(c, out.AccessToken, out.RefreshToken, out.ExpiresIn)
}

// Refresh handles POST /auth/refresh.
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
// Returns 204 No Content on success.
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

// handleUseCaseError maps domain errors to appropriate HTTP status codes.
// Generic messages are returned to clients to prevent information enumeration.
func (h *AuthHandler) handleUseCaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUserNotFound):
		writeError(c, http.StatusUnauthorized, "invalid email or password")

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
