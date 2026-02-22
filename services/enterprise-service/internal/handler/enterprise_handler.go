package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

type EnterpriseHandler struct {
	usecase domain.EnterpriseUsecase
}

func NewEnterpriseHandler(uc domain.EnterpriseUsecase) *EnterpriseHandler {
	return &EnterpriseHandler{usecase: uc}
}

type registerEnterpriseRequest struct {
	Slug          string `json:"slug" binding:"required"`
	DisplayName   string `json:"displayName" binding:"required"`
	LegalName     string `json:"legalName" binding:"required"`
	ContactEmail  string `json:"contactEmail" binding:"required,email"`
	OwnerEmail    string `json:"ownerEmail" binding:"required,email"`
	OwnerPassword string `json:"ownerPassword" binding:"required,min=8"`
}

func (h *EnterpriseHandler) Register(c *gin.Context) {
	var req registerEnterpriseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	enterprise := &domain.Enterprise{
		Slug:         req.Slug,
		DisplayName:  req.DisplayName,
		LegalName:    req.LegalName,
		ContactEmail: req.ContactEmail,
	}

	owner := &domain.User{
		Email:        req.OwnerEmail,
		PasswordHash: req.OwnerPassword,
	}

	created, err := h.usecase.RegisterEnterprise(c.Request.Context(), enterprise, owner)
	if err != nil {
		switch err {
		case domain.ErrSlugAlreadyExists:
			writeError(c, http.StatusConflict, "enterprise slug already exists")
		case domain.ErrEmailAlreadyExists:
			writeError(c, http.StatusConflict, "owner email already exists")
		default:
			writeError(c, http.StatusInternalServerError, "failed to register enterprise")
		}
		return
	}

	writeJSON(c, http.StatusCreated, created)
}

func (h *EnterpriseHandler) Approve(c *gin.Context) {
	idStr := c.Param("enterpriseId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	// For now, we assume admin ID is in context or passed via header
	apiAdminID := getRequestUserID(c)

	if err := h.usecase.ApproveEnterprise(c.Request.Context(), id, apiAdminID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to approve enterprise")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Suspend(c *gin.Context) {
	idStr := c.Param("enterpriseId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	apiAdminID := getRequestUserID(c)

	if err := h.usecase.SuspendEnterprise(c.Request.Context(), id, apiAdminID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to suspend enterprise")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Delete(c *gin.Context) {
	idStr := c.Param("enterpriseId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	apiAdminID := getRequestUserID(c)

	if err := h.usecase.DeleteEnterprise(c.Request.Context(), id, apiAdminID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to delete enterprise")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Get(c *gin.Context) {
	idStr := c.Param("enterpriseId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	enterprise, err := h.usecase.GetEnterprise(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrEnterpriseNotFound {
			writeError(c, http.StatusNotFound, "enterprise not found")
		} else {
			writeError(c, http.StatusInternalServerError, "failed to get enterprise")
		}
		return
	}

	writeJSON(c, http.StatusOK, enterprise)
}

func (h *EnterpriseHandler) Update(c *gin.Context) {
	idStr := c.Param("enterpriseId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	var e domain.Enterprise
	if err := c.ShouldBindJSON(&e); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	e.ID = id

	apiAdminID := getRequestUserID(c)

	if err := h.usecase.UpdateEnterprise(c.Request.Context(), &e, apiAdminID); err != nil {
		if err == domain.ErrEnterpriseNotFound {
			writeError(c, http.StatusNotFound, "enterprise not found")
		} else {
			writeError(c, http.StatusInternalServerError, "failed to update enterprise")
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func getRequestUserID(c *gin.Context) uuid.UUID {
	// TODO: Extract from context (middleware-set) or header
	// For now returning a dummy UUID if not found
	return uuid.Nil
}
