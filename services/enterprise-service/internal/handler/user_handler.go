package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// UserHandler manages enterprise users.
type UserHandler struct {
	usecase domain.UserUsecase
}

func NewUserHandler(uc domain.UserUsecase) *UserHandler {
	return &UserHandler{usecase: uc}
}

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

func (h *UserHandler) ListUsers(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	page, limit := ParsePagination(c)
	users, total, err := h.usecase.ListEnterpriseUsers(c.Request.Context(), id, page, limit)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	writeJSON(c, http.StatusOK, domain.PaginatedResult[*domain.User]{
		Items: users,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

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
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
