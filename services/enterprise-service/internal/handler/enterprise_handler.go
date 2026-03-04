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

// Register creates a new enterprise and owner account.
//
//	@Summary		Register enterprise
//	@Description	Register a new enterprise with an owner account.
//	@Tags			enterprise
//	@Accept			json
//	@Produce		json
//	@Param			body	body		EnterpriseRegisterRequest	true	"Registration payload"
//	@Success		201		{object}	domain.Enterprise
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/enterprises [post]
func (h *EnterpriseHandler) Register(c *gin.Context) {
	var req EnterpriseRegisterRequest
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

// Get returns one enterprise by ID.
//
//	@Summary		Get enterprise
//	@Description	Get enterprise details by enterprise ID.
//	@Tags			enterprise
//	@Produce		json
//	@Param			enterpriseId	path		string	true	"Enterprise ID (UUID)"
//	@Success		200			{object}	domain.Enterprise
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId} [get]
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

// Update updates enterprise profile fields.
//
//	@Summary		Update enterprise
//	@Description	Update enterprise profile fields.
//	@Tags			enterprise
//	@Accept			json
//	@Param			enterpriseId	path	string			true	"Enterprise ID (UUID)"
//	@Param			body			body	domain.Enterprise	true	"Enterprise patch payload"
//	@Success		204			{string}	string			"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId} [patch]
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

// Approve approves a pending enterprise.
//
//	@Summary		Approve enterprise
//	@Description	Approve enterprise onboarding request.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/approve [post]
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

// Suspend suspends an enterprise.
//
//	@Summary		Suspend enterprise
//	@Description	Suspend an active enterprise.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/suspend [post]
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

// Delete soft-deletes an enterprise.
//
//	@Summary		Delete enterprise
//	@Description	Soft-delete enterprise and start retention period.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId} [delete]
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

// List returns paginated enterprises.
//
//	@Summary		List enterprises
//	@Description	List enterprises with optional status and search filters.
//	@Tags			enterprise
//	@Produce		json
//	@Param			status				query	string	false	"Enterprise status"
//	@Param			subscription_status	query	string	false	"Subscription status"
//	@Param			search				query	string	false	"Search by slug or display name"
//	@Param			page				query	int	false	"Page number"
//	@Param			limit				query	int	false	"Page size"
//	@Success		200				{object}	EnterpriseListResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/enterprises [get]
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

// GetBySlug returns one enterprise by slug.
//
//	@Summary		Get enterprise by slug
//	@Description	Resolve enterprise details using slug.
//	@Tags			enterprise
//	@Produce		json
//	@Param			slug	path		string	true	"Enterprise slug"
//	@Success		200		{object}	domain.Enterprise
//	@Failure		404		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/enterprises/slug/{slug} [get]
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

// GetMe returns caller's enterprise details.
//
//	@Summary		Get my enterprise
//	@Description	Get enterprise inferred from X-Enterprise-ID header.
//	@Tags			enterprise
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Caller enterprise ID (UUID)"
//	@Success		200				{object}	domain.Enterprise
//	@Failure		400				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/enterprises/me [get]
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

// UpdateBranding updates enterprise branding attributes.
//
//	@Summary		Update branding
//	@Description	Update logo and brand color values.
//	@Tags			enterprise
//	@Accept			json
//	@Param			enterpriseId	path	string					true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string					false	"Actor user ID (UUID)"
//	@Param			body			body	domain.UpdateBrandingRequest	true	"Branding patch"
//	@Success		204			{string}	string					"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/branding [patch]
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

// UpdateSettings partially updates enterprise settings.
//
//	@Summary		Update settings
//	@Description	Merge/patch enterprise settings object.
//	@Tags			enterprise
//	@Accept			json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Param			body			body	object	true	"Settings patch object"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/settings [patch]
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

// Reactivate reactivates a suspended enterprise.
//
//	@Summary		Reactivate enterprise
//	@Description	Reactivate enterprise from suspended state.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/reactivate [post]
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

// Restore restores a soft-deleted enterprise during retention.
//
//	@Summary		Restore enterprise
//	@Description	Restore a soft-deleted enterprise.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/restore [post]
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

// HardDelete permanently deletes an enterprise after retention checks.
//
//	@Summary		Hard delete enterprise
//	@Description	Permanently delete enterprise data.
//	@Tags			enterprise
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/permanent [delete]
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

// GetStatus returns lifecycle and subscription status for an enterprise.
//
//	@Summary		Get enterprise status
//	@Description	Get lifecycle and subscription status view.
//	@Tags			enterprise
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Success		200			{object}	domain.EnterpriseStatusResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/status [get]
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

// ValidateDomain validates enterprise custom domain DNS settings.
//
//	@Summary		Validate custom domain
//	@Description	Validate TXT/CNAME records for enterprise custom domain.
//	@Tags			enterprise
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Success		200			{object}	domain.DomainValidationResult
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		422			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/validate-domain [post]
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

// GetSummary returns high-level enterprise metrics.
//
//	@Summary		Get enterprise summary
//	@Description	Get compact operational summary for an enterprise.
//	@Tags			enterprise
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Success		200			{object}	domain.EnterpriseSummary
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/summary [get]
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

// GetAuditLogs returns paginated enterprise audit logs.
//
//	@Summary		Get audit logs
//	@Description	List paginated audit logs for enterprise actions.
//	@Tags			enterprise
//	@Produce		json
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			page			query	int	false	"Page number"
//	@Param			limit			query	int	false	"Page size"
//	@Success		200			{object}	AuditLogListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		409			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/audit-logs [get]
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
