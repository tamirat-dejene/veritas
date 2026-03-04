package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"go.uber.org/zap"
)

type CandidateHandler struct {
	uc     domain.CandidateUseCase
	logger *zap.Logger
}

func NewCandidateHandler(uc domain.CandidateUseCase, logger *zap.Logger) *CandidateHandler {
	return &CandidateHandler{
		uc:     uc,
		logger: logger,
	}
}

// Ensure middleware sets "x-enterprise-id" into context. Usually API gateway does this.
func getEnterpriseID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("enterprise_id")
	if !exists {
		// Fallback to header if running directly without exact middleware mapping
		headerVal := c.GetHeader("X-Enterprise-Id")
		if headerVal != "" {
			return uuid.Parse(headerVal)
		}
		return uuid.Nil, domain.ErrUnauthorizedAccess
	}
	return uuid.Parse(val.(string))
}

// Create registers a single candidate profile for the current enterprise.
//
//	@Summary		Create candidate
//	@Description	Create one candidate profile under the caller enterprise.
//	@Tags			candidate
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string				false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			body				body		CandidateCreateRequest	true	"Candidate payload"
//	@Success		201				{object}	CandidateResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		409				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates [post]
func (h *CandidateHandler) Create(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	var req CandidateCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	candidate := &domain.CandidateProfile{
		EnterpriseID:     entID,
		ExternalID:       req.ExternalID,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
		FaceReferenceURL: req.FaceReferenceURL,
		IsActive:         true,
	}

	created, err := h.uc.CreateCandidate(c.Request.Context(), candidate)
	if err != nil {
		if err == domain.ErrDuplicateExternalID {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create candidate"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// BulkUpload creates candidates from a CSV file upload.
//
//	@Summary		Bulk upload candidates
//	@Description	Create many candidate profiles from a CSV file (max 5MB).
//	@Tags			candidate
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			file				formData	file	true	"CSV file"
//	@Success		201				{object}	BulkUploadResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		413				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates/bulk [post]
func (h *CandidateHandler) BulkUpload(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'file' field in multipart form"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File size exceeds 5MB limit"})
		return
	}

	csvData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	count, err := h.uc.BulkUpload(c.Request.Context(), entID, csvData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bulk upload successful", "count": count})
}

// List returns all candidates for the caller enterprise.
//
//	@Summary		List candidates
//	@Description	List candidate profiles for the caller enterprise.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Success		200				{object}	CandidateListResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates [get]
func (h *CandidateHandler) List(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	list, err := h.uc.GetCandidates(c.Request.Context(), entID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch candidates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Get retrieves a candidate by candidate ID.
//
//	@Summary		Get candidate
//	@Description	Get one candidate profile by ID for the caller enterprise.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	CandidateResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates/{candidateId} [get]
func (h *CandidateHandler) Get(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate ID format"})
		return
	}

	candidate, err := h.uc.GetCandidate(c.Request.Context(), id, entID)
	if err != nil {
		if err == domain.ErrCandidateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Candidate not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch candidate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": candidate})
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
//	@Param			body				body		CandidateUpdateRequest	true	"Updated candidate payload"
//	@Success		200				{object}	MessageResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates/{candidateId} [patch]
func (h *CandidateHandler) Update(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate ID format"})
		return
	}

	var req CandidateUpdateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	candidate := &domain.CandidateProfile{
		ID:               id,
		EnterpriseID:     entID,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
		FaceReferenceURL: req.FaceReferenceURL,
		IsActive:         req.IsActive,
	}

	if err := h.uc.UpdateCandidate(c.Request.Context(), candidate); err != nil {
		if err == domain.ErrCandidateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Candidate not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update candidate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Candidate updated"})
}

// Deactivate disables a candidate profile.
//
//	@Summary		Deactivate candidate
//	@Description	Soft-deactivate a candidate profile by ID.
//	@Tags			candidate
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			candidateId		path	string	true	"Candidate ID (UUID)"
//	@Success		200				{object}	MessageResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/candidates/{candidateId}/deactivate [patch]
func (h *CandidateHandler) Deactivate(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("candidateId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate ID format"})
		return
	}

	if err := h.uc.DeactivateCandidate(c.Request.Context(), id, entID); err != nil {
		if err == domain.ErrCandidateNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Candidate not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate candidate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Candidate deactivated"})
}
