package http

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

func (h *CandidateHandler) Create(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	var req struct {
		ExternalID       string  `json:"externalId" binding:"required"`
		FirstName        string  `json:"firstName" binding:"required"`
		LastName         string  `json:"lastName" binding:"required"`
		Email            *string `json:"email"`
		FaceReferenceURL *string `json:"faceReferenceUrl"`
	}

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

	var req struct {
		FirstName        string  `json:"firstName" binding:"required"`
		LastName         string  `json:"lastName" binding:"required"`
		Email            *string `json:"email"`
		FaceReferenceURL *string `json:"faceReferenceUrl"`
		IsActive         bool    `json:"isActive"`
	}

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
