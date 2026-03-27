package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
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

// Enroll enrolls one or more candidates to an exam.
//
//	@Summary		Enroll candidates
//	@Description	Create enrollments for an exam and return generated raw access tokens.
//	@Tags			enrollment
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string			false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path		string			true	"Exam ID (UUID)"
//	@Param			body				body		dto.EnrollmentRequest	true	"Enrollment payload"
//	@Success		201				{object}	dto.EnrollmentCreateResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/enrollments [post]
func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Enterprise ID missing in request", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID format"})
		return
	}

	var req dto.EnrollmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Invalid enrollment request", zap.Error(err), zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokens, err := h.uc.EnrollCandidates(c.Request.Context(), entID, examID, req.CandidateIDs, req.MaxAttempts, req.TokenExpiresAt)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Error("Failed to enroll candidates", zap.Error(err), zap.String("examID", examID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enroll candidates"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Enrolled successfully", "enrollmentTokens": tokens})
}

// ListByExam lists enrollments for an exam with pagination.
//
//	@Summary		List enrollments by exam
//	@Description	List exam enrollments for the caller enterprise with pagination.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			page			query	int		false	"Page number (default 1)"
//	@Param			limit			query	int		false	"Page size (default 10, max 1000)"
//	@Param			sort			query	string	false	"Sort field: created_at|attempts_used"
//	@Param			sort_dir		query	string	false	"Sort direction: asc|desc (default desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.ExamEnrollment]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/enrollments [get]
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

	params := pagination.ParseGin(c)
	list, total, err := h.uc.GetEnrollmentsForExam(c.Request.Context(), examID, entID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrollments"})
		return
	}

	c.JSON(http.StatusOK, pagination.NewPaginatedResponse(list, total, params))
}

// Get fetches enrollment details by enrollment ID.
//
//	@Summary		Get enrollment
//	@Description	Get one enrollment by ID.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			enrollmentId		path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.EnrollmentResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId} [get]
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

// RegenerateToken rotates the enrollment access token.
//
//	@Summary		Regenerate enrollment token
//	@Description	Regenerate and return a new raw token for an enrollment.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			enrollmentId		path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.EnrollmentTokenResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/regenerate-token [post]
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

// Revoke revokes an enrollment.
//
//	@Summary		Revoke enrollment
//	@Description	Revoke an enrollment and prevent future use.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			enrollmentId		path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/revoke [patch]
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

// ResetAttempts resets enrollment attempt counters.
//
//	@Summary		Reset enrollment attempts
//	@Description	Reset attempts used to zero for an enrollment.
//	@Tags			enrollment
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			enrollmentId		path	string	true	"Enrollment ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/enrollments/{enrollmentId}/reset-attempts [post]
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
