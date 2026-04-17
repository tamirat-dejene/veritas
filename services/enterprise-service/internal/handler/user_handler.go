package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
)

// UserHandler manages enterprise users.
type UserHandler struct {
	usecase domain.UserUsecase
}

func NewUserHandler(uc domain.UserUsecase) *UserHandler {
	return &UserHandler{usecase: uc}
}

// CreateUser creates a new user under an enterprise.
//
//	@Summary		Create enterprise user
//	@Description	Create a user account scoped to an enterprise.
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			enterpriseId	path	string				true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string				false	"Actor user ID (UUID)"
//	@Param			body			body	domain.CreateUserRequest	true	"Create user payload"
//	@Success		201			{object}	domain.User
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	callerID, _ := GetCallerID(c)
	user, err := h.usecase.CreateEnterpriseUser(c.Request.Context(), id, req, callerID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	writeJSON(c, http.StatusCreated, user)
}

// ListUsers lists enterprise users with pagination.
//
//	@Summary		List enterprise users
//	@Description	List users belonging to the specified enterprise.
//	@Tags			user
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			page			query	int		false	"Page number"
//	@Param			limit			query	int		false	"Page size"
//	@Param			sort			query	string	false	"Sort field (email, first_name, last_name, role, created_at)"
//	@Param			sort_dir		query	string	false	"Sort direction (asc, desc)"
//	@Success		200			{object}	pagination.PaginatedResponse[domain.User]
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	params := pagination.ParseGin(c)
	users, total, err := h.usecase.ListEnterpriseUsers(c.Request.Context(), id, params)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	writeJSON(c, http.StatusOK, pagination.NewPaginatedResponse(users, int64(total), params))
}

// GetUser gets one enterprise user by ID.
//
//	@Summary		Get enterprise user
//	@Description	Get a specific enterprise user by user ID.
//	@Tags			user
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			userId			path	string	true	"User ID (UUID)"
//	@Success		200			{object}	domain.User
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users/{userId} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	enterpriseID, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	userID, ok := ParseUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	user, err := h.usecase.GetEnterpriseUser(c.Request.Context(), enterpriseID, userID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	writeJSON(c, http.StatusOK, user)
}

// UpdateUser updates an enterprise user.
//
//	@Summary		Update enterprise user
//	@Description	Update profile and role fields for an enterprise user.
//	@Tags			user
//	@Accept			json
//	@Param			enterpriseId	path	string				true	"Enterprise ID (UUID)"
//	@Param			userId			path	string				true	"User ID (UUID)"
//	@Param			X-User-ID	header	string				false	"Actor user ID (UUID)"
//	@Param			body			body	domain.UpdateUserRequest	true	"Update user payload"
//	@Success		204			{string}	string				"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users/{userId} [patch]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	enterpriseID, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	userID, ok := ParseUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	var req domain.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.UpdateEnterpriseUser(c.Request.Context(), enterpriseID, userID, req, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// DeactivateUser deactivates an enterprise user account.
//
//	@Summary		Deactivate enterprise user
//	@Description	Deactivate user account without permanent deletion.
//	@Tags			user
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			userId			path	string	true	"User ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users/{userId}/deactivate [patch]
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	enterpriseID, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	userID, ok := ParseUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.DeactivateEnterpriseUser(c.Request.Context(), enterpriseID, userID, callerID); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ResetPassword resets an enterprise user's password and returns a temporary password.
//
//	@Summary		Reset user password
//	@Description	Reset user password and return temporary password for secure handoff.
//	@Tags			user
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			userId			path	string	true	"User ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		200			{object}	ResetPasswordResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users/{userId}/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	enterpriseID, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	userID, ok := ParseUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	callerID, _ := GetCallerID(c)
	tempPwd, err := h.usecase.ResetUserPassword(c.Request.Context(), enterpriseID, userID, callerID)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	// Return temp password so admin can share it securely
	writeJSON(c, http.StatusOK, gin.H{"temporary_password": tempPwd})
}

// ChangePassword allows a user to change their own password.
//
//	@Summary		Change user password
//	@Description	Self-service password change for authenticated users.
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			enterpriseId	path	string						true	"Enterprise ID (UUID)"
//	@Param			userId			path	string						true	"User ID (UUID)"
//	@Param			X-User-ID	header	string						true	"Actor user ID (UUID)"
//	@Param			body			body	domain.ChangePasswordRequest	true	"Change password payload"
//	@Success		204			{string}	string						"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/users/{userId}/change-password [post]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := ParseUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	callerID, ok := GetCallerID(c)
	if !ok || callerID != userID {
		writeError(c, http.StatusUnauthorized, "unauthorized to change password for another user")
		return
	}

	var req domain.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.usecase.ChangePassword(c.Request.Context(), userID, req); err != nil {
		h.handleErr(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *UserHandler) handleErr(c *gin.Context, err error) {
	switch err {
	case domain.ErrUserNotFound:
		writeError(c, http.StatusNotFound, "user not found")
	case domain.ErrEnterpriseNotFound:
		writeError(c, http.StatusNotFound, "enterprise not found")
	case domain.ErrEmailAlreadyExists:
		writeError(c, http.StatusConflict, "email already exists")
	case domain.ErrInvalidRole:
		writeError(c, http.StatusBadRequest, "role not allowed for enterprise users")
	case domain.ErrInvalidCredentials:
		writeError(c, http.StatusUnauthorized, "invalid credentials")
	default:
		zap.L().Error("Unhandled user error", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}

// Internal endpoints for service-to-service communication

func (h *UserHandler) GetByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		writeError(c, http.StatusBadRequest, "email query parameter is required")
		return
	}
	user, err := h.usecase.GetByEmail(c.Request.Context(), email)
	if err != nil {
		h.handleErr(c, err)
		return
	}

	writeJSON(c, http.StatusOK, user)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	user, err := h.usecase.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleErr(c, err)
		return
	}

	writeJSON(c, http.StatusOK, user)
}

func (h *UserHandler) RecordLoginSuccess(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	var req struct {
		IP        string `json:"ip"`
		UserAgent string `json:"user_agent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.RecordLoginSuccess(c.Request.Context(), id, req.IP, req.UserAgent); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) RecordLoginFailure(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	var req struct {
		LockedUntil *time.Time `json:"locked_until"`
		FailedLoginAttempts int        `json:"failed_login_attempts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.usecase.RecordLoginFailure(c.Request.Context(), id, req.LockedUntil, req.FailedLoginAttempts); err != nil {
		h.handleErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
