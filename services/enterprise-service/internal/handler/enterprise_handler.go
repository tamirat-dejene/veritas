package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
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
		zap.L().Error("failed to register enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to get enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to update enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to approve enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to suspend enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to delete enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
//	@Param			search				query	string	false	"Search by slug or display name"
//	@Param			page				query	int		false	"Page number"
//	@Param			limit				query	int		false	"Page size"
//	@Param			sort				query	string	false	"Sort field (display_name, slug, status, created_at)"
//	@Param			sort_dir			query	string	false	"Sort direction (asc, desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.Enterprise]
//	@Failure		500				{object}	ErrorResponse
//	@Router			/enterprises [get]
func (h *EnterpriseHandler) List(c *gin.Context) {
	filter := domain.EnterpriseFilter{}
	if s := c.Query("status"); s != "" {
		st := domain.EnterpriseStatus(s)
		filter.Status = &st
	}
	filter.Search = c.Query("search")
	filter.Params = pagination.ParseGin(c)

	items, total, err := h.usecase.ListEnterprises(c.Request.Context(), filter)
	if err != nil {
		zap.L().Error("failed to list enterprises", zap.Error(err))
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, pagination.NewPaginatedResponse(items, int64(total), filter.Params))
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
	adminID, ok := GetCallerID(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthenticated")
		return
	}

	enterprise, err := h.usecase.GetEnterpriseBySlug(c.Request.Context(), slug, adminID)
	if err != nil {
		zap.L().Error("failed to get enterprise by slug", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to get enterprise", zap.Error(err))
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, enterprise)
}

// ─── Branding & Settings ──────────────────────────────────────────────────────

// UpdateBranding updates enterprise branding attributes.
//
//	@Summary		Update branding
//	@Description	Update brand color values (logo updation is moved to an independent endpoint).
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
		zap.L().Error("failed to update branding", zap.Error(err))
		h.handleEnterpriseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// UploadLogo uploads a new enterprise logo.
//
//	@Summary		Upload logo
//	@Description	Upload a new enterprise logo image. The logo should be in png, jpg, or jpeg format and the file size should not exceed 3MB.
//	@Tags			enterprise
//	@Accept			multipart/form-data
//	@Param			enterpriseId	path	string	true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string	false	"Actor user ID (UUID)"
//	@Param			logo			formData	file	true	"Logo file (png, jpg, jpeg)"
//	@Success		200			{object}	UploadLogoResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/enterprises/{enterpriseId}/logo [post]
func (h *EnterpriseHandler) UploadLogo(c *gin.Context) {
	id, ok := ParseEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	// 1. Limit file size (e.g., 3MB)
	const maxFileSize = 3 * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	file, _, err := c.Request.FormFile("logo")
	if err != nil {
		if err.Error() == "http: request body too large" {
			writeError(c, http.StatusRequestEntityTooLarge, "file too large (max 3MB)")
		} else {
			writeError(c, http.StatusBadRequest, "no logo file provided")
		}
		return
	}
	defer file.Close()

	// 2. Validate file type
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to read file header")
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to reset file pointer")
		return
	}

	contentType := http.DetectContentType(buff)
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
	}

	if !allowedTypes[contentType] {
		writeError(c, http.StatusBadRequest, "invalid file type (png, jpg, jpeg only)")
		return
	}

	callerID, _ := GetCallerID(c)
	// Use a standard filename based on enterprise ID to ensure overwriting of existing logos
	logoFileName := fmt.Sprintf("logo_%s", id.String())
	url, err := h.usecase.UploadLogo(c.Request.Context(), id, logoFileName, file, callerID)

	if err != nil {
		zap.L().Error("failed to upload logo", zap.Error(err))
		h.handleEnterpriseError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, UploadLogoResponse{LogoURL: url})
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
		zap.L().Error("failed to update settings", zap.Error(err))
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
		zap.L().Error("failed to reactivate enterprise", zap.Error(err))
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
		zap.L().Error("failed to restore enterprise", zap.Error(err))
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
		zap.L().Error("failed to hard delete enterprise", zap.Error(err))
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
		zap.L().Error("failed to get enterprise status", zap.Error(err))
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
		zap.L().Error("failed to validate custom domain", zap.Error(err))
		h.handleEnterpriseError(c, err)
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
		zap.L().Error("failed to get enterprise summary", zap.Error(err))
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
//	@Param			page			query	int		false	"Page number"
//	@Param			limit			query	int		false	"Page size"
//	@Param			sort			query	string	false	"Sort field (event, actor_role, created_at)"
//	@Param			sort_dir		query	string	false	"Sort direction (asc, desc)"
//	@Success		200			{object}	pagination.PaginatedResponse[domain.AuditLog]
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
	params := pagination.ParseGin(c)
	logs, total, err := h.usecase.GetAuditLogs(c.Request.Context(), id, params)
	if err != nil {
		zap.L().Error("failed to get audit logs", zap.Error(err))
		h.handleEnterpriseError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, pagination.NewPaginatedResponse(logs, int64(total), params))
}

// ─── Error helper ─────────────────────────────────────────────────────────────

func (h *EnterpriseHandler) handleEnterpriseError(c *gin.Context, err error) {
	log := zap.L().With(zap.String("path", c.FullPath()), zap.Error(err))
	switch err {
	case domain.ErrEnterpriseNotFound:
		log.Warn("enterprise not found")
		writeError(c, http.StatusNotFound, "enterprise not found")
	case domain.ErrInvalidStatus:
		log.Warn("invalid status transition")
		writeError(c, http.StatusConflict, "invalid status transition")
	case domain.ErrRetentionActive:
		log.Warn("retention period has not expired yet")
		writeError(c, http.StatusConflict, "retention period has not expired yet")
	case domain.ErrForbidden:
		log.Warn("forbidden")
		writeError(c, http.StatusForbidden, "forbidden")
	case domain.ErrDomainValidation:
		log.Warn("domain validation error")
		writeError(c, http.StatusUnprocessableEntity, "domain validation error")
	case domain.ErrSlugAlreadyExists:
		log.Warn("enterprise slug already exists")
		writeError(c, http.StatusConflict, "enterprise slug already exists")
	case domain.ErrEmailAlreadyExists:
		log.Warn("owner email already exists")
		writeError(c, http.StatusConflict, "owner email already exists")
	case domain.ErrUserNotFound:
		log.Warn("user not found")
		writeError(c, http.StatusNotFound, "user not found")
	default:
		log.Error("unhandled enterprise error")
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}
