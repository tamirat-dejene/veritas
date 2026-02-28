package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"go.uber.org/zap"
)

type EnrollmentHandler struct {
	uc     domain.EnrollmentUseCase
	logger *zap.Logger
}

func NewEnrollmentHandler(uc domain.EnrollmentUseCase, logger *zap.Logger) *EnrollmentHandler {
	return &EnrollmentHandler{
		uc:     uc,
		logger: logger,
	}
}

func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID format"})
		return
	}

	var req struct {
		CandidateIDs     []uuid.UUID `json:"candidateIds" binding:"required,min=1"`
		InvitationMethod string      `json:"invitationMethod" binding:"required"`
		MaxAttempts      int         `json:"maxAttempts" binding:"required,min=1"`
		TokenExpiresAt   time.Time   `json:"tokenExpiresAt" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.uc.EnrollCandidates(c.Request.Context(), entID, examID, req.CandidateIDs, req.InvitationMethod, req.MaxAttempts, req.TokenExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll candidates"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Enrolled successfully", "rawTokens": tokens})
}

func (h *EnrollmentHandler) ListByExam(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID format"})
		return
	}

	list, err := h.uc.GetEnrollmentsForExam(c.Request.Context(), examID, entID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *EnrollmentHandler) Get(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("enrollmentId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID format"})
		return
	}

	enr, err := h.uc.GetEnrollment(c.Request.Context(), id, entID)
	if err != nil {
		if err == domain.ErrEnrollmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": enr})
}

func (h *EnrollmentHandler) RegenerateToken(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("enrollmentId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID format"})
		return
	}

	newToken, err := h.uc.RegenerateToken(c.Request.Context(), id, entID)
	if err != nil {
		if err == domain.ErrEnrollmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token regenerated", "rawToken": newToken})
}

func (h *EnrollmentHandler) Revoke(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("enrollmentId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID format"})
		return
	}

	if err := h.uc.RevokeEnrollment(c.Request.Context(), id, entID); err != nil {
		if err == domain.ErrEnrollmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke enrollment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Enrollment revoked"})
}

func (h *EnrollmentHandler) ResetAttempts(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	idParam := c.Param("enrollmentId")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid enrollment ID format"})
		return
	}

	if err := h.uc.ResetAttempts(c.Request.Context(), id, entID); err != nil {
		if err == domain.ErrEnrollmentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Enrollment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset attempts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Enrollment attempts reset"})
}
