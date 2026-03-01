package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// EnterpriseHandler handles enterprise-level HTTP requests.
type EnterpriseHandler struct {
	usecase domain.EnterpriseUsecase
}

func NewEnterpriseHandler(uc domain.EnterpriseUsecase) *EnterpriseHandler {
	return &EnterpriseHandler{usecase: uc}
}

// ─── Existing endpoints ───────────────────────────────────────────────────────

type registerEnterpriseRequest struct {
	Slug          string `json:"slug"          binding:"required"`
	DisplayName   string `json:"displayName"   binding:"required"`
	LegalName     string `json:"legalName"     binding:"required"`
	ContactEmail  string `json:"contactEmail"  binding:"required,email"`
	OwnerEmail    string `json:"ownerEmail"    binding:"required,email"`
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

func (h *EnterpriseHandler) Get(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
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
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	var e domain.Enterprise
	if err := c.ShouldBindJSON(&e); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	e.ID = id
	callerID, _ := GetCallerID(c)
	if err := h.usecase.UpdateEnterprise(c.Request.Context(), &e, callerID); err != nil {
		if err == domain.ErrEnterpriseNotFound {
			writeError(c, http.StatusNotFound, "enterprise not found")
		} else {
			writeError(c, http.StatusInternalServerError, "failed to update enterprise")
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Approve(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.ApproveEnterprise(c.Request.Context(), id, callerID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to approve enterprise")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Suspend(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.SuspendEnterprise(c.Request.Context(), id, callerID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to suspend enterprise")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Delete(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.DeleteEnterprise(c.Request.Context(), id, callerID); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to delete enterprise")
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Discovery & Listing ──────────────────────────────────────────────────────

func (h *EnterpriseHandler) List(c *gin.Context) {
	filter := domain.EnterpriseFilter{}
	if s := c.Query("status"); s != "" {
		st := domain.EnterpriseStatus(s)
		filter.Status = &st
	}
	if s := c.Query("subscription_status"); s != "" {
		ss := domain.SubscriptionStatus(s)
		filter.SubscriptionStatus = &ss
	}
	filter.Search = c.Query("search")
	filter.Page, filter.Limit = ParsePagination(c)

	items, total, err := h.usecase.ListEnterprises(c.Request.Context(), filter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to list enterprises")
		return
	}
	writeJSON(c, http.StatusOK, domain.PaginatedResult[*domain.Enterprise]{
		Items: items,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	})
}

func (h *EnterpriseHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	enterprise, err := h.usecase.GetEnterpriseBySlug(c.Request.Context(), slug)
	if err != nil {
		if err == domain.ErrEnterpriseNotFound {
			writeError(c, http.StatusNotFound, "enterprise not found")
		} else {
			writeError(c, http.StatusInternalServerError, "failed to get enterprise by slug")
		}
		return
	}
	writeJSON(c, http.StatusOK, enterprise)
}

func (h *EnterpriseHandler) GetMe(c *gin.Context) {
	enterpriseID, ok := GetCallerEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "X-Enterprise-ID header missing or invalid")
		return
	}
	enterprise, err := h.usecase.GetMyEnterprise(c.Request.Context(), enterpriseID)
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

// ─── Branding & Settings ──────────────────────────────────────────────────────

func (h *EnterpriseHandler) UpdateBranding(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	var req domain.UpdateBrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.UpdateBranding(c.Request.Context(), id, req, callerID); err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) UpdateSettings(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.UpdateSettings(c.Request.Context(), id, patch, callerID); err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Lifecycle & Governance ───────────────────────────────────────────────────

func (h *EnterpriseHandler) Reactivate(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.ReactivateEnterprise(c.Request.Context(), id, callerID); err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) Restore(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.RestoreEnterprise(c.Request.Context(), id, callerID); err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *EnterpriseHandler) HardDelete(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	if err := h.usecase.HardDeleteEnterprise(c.Request.Context(), id, callerID); err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─── Status, Domain, Audit ────────────────────────────────────────────────────

func (h *EnterpriseHandler) GetStatus(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	resp, err := h.usecase.GetEnterpriseStatus(c.Request.Context(), id)
	if err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, resp)
}

func (h *EnterpriseHandler) ValidateDomain(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	callerID, _ := GetCallerID(c)
	result, err := h.usecase.ValidateCustomDomain(c.Request.Context(), id, callerID)
	if err != nil {
		if err == domain.ErrDomainValidation {
			writeError(c, http.StatusUnprocessableEntity, "no custom domain configured for this enterprise")
		} else {
			h.handleEnterpriseError(c, err)
		}
		return
	}
	writeJSON(c, http.StatusOK, result)
}

func (h *EnterpriseHandler) GetSummary(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	summary, err := h.usecase.GetEnterpriseSummary(c.Request.Context(), id)
	if err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, summary)
}

func (h *EnterpriseHandler) GetAuditLogs(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}
	page, limit := ParsePagination(c)
	logs, total, err := h.usecase.GetAuditLogs(c.Request.Context(), id, page, limit)
	if err != nil {
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, domain.PaginatedResult[*domain.AuditLog]{
		Items: logs,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// ─── Error helper ─────────────────────────────────────────────────────────────

func (h *EnterpriseHandler) handleEnterpriseError(c *gin.Context, err error) {
	switch err {
	case domain.ErrEnterpriseNotFound:
		writeError(c, http.StatusNotFound, "enterprise not found")
	case domain.ErrInvalidStatus:
		writeError(c, http.StatusConflict, "invalid status transition")
	case domain.ErrRetentionActive:
		writeError(c, http.StatusConflict, "retention period has not expired yet")
	case domain.ErrForbidden:
		writeError(c, http.StatusForbidden, "forbidden")
	case domain.ErrDomainValidation:
		writeError(c, http.StatusUnprocessableEntity, "domain validation error")
	default:
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}

// getRequestUserID kept for compat (replaced by GetCallerID).
func getRequestUserID(c *gin.Context) uuid.UUID {
	id, _ := GetCallerID(c)
	return id
}
