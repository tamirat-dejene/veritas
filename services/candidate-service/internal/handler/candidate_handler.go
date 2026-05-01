package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type CandidateHandler struct {
	uc domain.CandidateUseCase
}

func NewCandidateHandler(uc domain.CandidateUseCase) *CandidateHandler {
	return &CandidateHandler{
		uc: uc,
	}
}

// Create registers a single candidate profile for the current enterprise.
//
//	@Summary		Create candidate
//	@Description	Create one candidate profile under the caller enterprise.
//	@Tags			candidate
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string				false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			body				body		dto.CandidateCreateRequest	true	"Candidate payload"
//	@Success		201				{object}	dto.CandidateResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		409				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates [post]
func (h *CandidateHandler) Create(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	var req dto.CandidateCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	candidate := &domain.CandidateProfile{
		EnterpriseID:     entID,
		ExternalID:       req.ExternalID,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
		IsActive:         true,
	}

	created, err := h.uc.CreateCandidate(c.Request.Context(), candidate)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CandidateResponse{Data: created})
}

// BulkUpload handles CSV upload for candidates.
//
//	@Summary		Bulk upload candidates
//	@Description	Create many candidate profiles from a CSV file (max 5MB).
//	@Description	The CSV should have the following columns in order:
//	@Description	external_id (required), first_name (required), last_name (required), email (optional).
//	@Description	The first row is expected to be a header and will be skipped.
//	@Tags			candidate
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			file				formData	file	true	"CSV file (max 5MB)"
//	@Success		201				{object}	dto.BulkUploadResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		413				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/bulk [post]
func (h *CandidateHandler) BulkUpload(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrMissingFile.Error()})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, dto.ErrorResponse{Error: domain.ErrFileTooLarge.Error()})
		return
	}

	csvData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: domain.ErrReadFailed.Error()})
		return
	}

	count, err := h.uc.BulkUpload(c.Request.Context(), entID, csvData)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.BulkUploadResponse{Message: "Bulk upload successful", Count: count})
}

// List returns a paginated list of candidates for the caller enterprise.
//
//	@Summary		List candidates
//	@Description	List candidate profiles for the caller enterprise with pagination.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			page			query	int		false	"Page number (default 1)"
//	@Param			limit			query	int		false	"Page size (default 10, max 1000)"
//	@Param			sort			query	string	false	"Sort field: created_at|first_name|last_name|external_id"
//	@Param			sort_dir		query	string	false	"Sort direction: asc|desc (default desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.CandidateProfile]
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates [get]
func (h *CandidateHandler) List(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	params := pagination.ParseGin(c)
	list, total, err := h.uc.GetCandidates(c.Request.Context(), entID, params)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pagination.NewPaginatedResponse(list, total, params))
}

// Get retrieves a candidate by candidate ID.
//
//	@Summary		Get candidate
//	@Description	Get one candidate profile by ID for the caller enterprise.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	dto.CandidateResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/{candidateId} [get]
func (h *CandidateHandler) Get(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	candidate, err := h.uc.GetCandidate(c.Request.Context(), id, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CandidateResponse{Data: candidate})
}

// Update modifies an existing candidate profile.
//
//	@Summary		Update candidate
//	@Description	Update candidate profile fields by ID.
//	@Tags			candidate
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string				false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path		string				true	"Candidate ID (UUID)"
//	@Param			body				body		dto.CandidateUpdateRequest	true	"Updated candidate payload"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/{candidateId} [patch]
func (h *CandidateHandler) Update(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	var req dto.CandidateUpdateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	candidate := &domain.CandidateProfile{
		ID:               id,
		EnterpriseID:     entID,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
	}

	if err := h.uc.UpdateCandidate(c.Request.Context(), candidate); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Candidate updated"})
}

// Deactivate disables a candidate profile.
//
//	@Summary		Deactivate candidate
//	@Description	Soft-deactivate a candidate profile by ID.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		403				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		409				{object}	dto.ErrorResponse
//	@Failure		413				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/{candidateId}/deactivate [patch]
func (h *CandidateHandler) Deactivate(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.DeactivateCandidate(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Candidate deactivated"})
}

// Activate enables a candidate profile.
//
//	@Summary		Activate candidate
//	@Description	Activate a candidate profile by ID.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		403				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		409				{object}	dto.ErrorResponse
//	@Failure		413				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/{candidateId}/activate [patch]
func (h *CandidateHandler) Activate(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.ActivateCandidate(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Candidate activated"})
}

// Delete permanently removes a candidate profile and all cascading data.
//
//	@Summary		Delete candidate
//	@Description	Permanently delete a candidate profile by ID and all associated enrollments/sessions.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		403				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/candidates/{candidateId} [delete]
func (h *CandidateHandler) Delete(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	if err := h.uc.DeleteCandidate(c.Request.Context(), id, entID); err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Candidate deleted"})
}

func (h *CandidateHandler) GetEmailsForExam(c *gin.Context) {
	examIDStr := c.Query("exam_id")
	enterpriseIDStr := c.Query("enterprise_id")

	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid exam_id"})
		return
	}

	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid enterprise_id"})
		return
	}

	emails, err := h.uc.GetEmailsByExamID(c.Request.Context(), examID, enterpriseID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CandidateEmailsResponse{Emails: emails})
}
