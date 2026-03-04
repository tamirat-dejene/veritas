package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"go.uber.org/zap"
)

type MonitoringHandler struct {
	uc     domain.MonitoringUseCase
	logger *zap.Logger
}

func NewMonitoringHandler(uc domain.MonitoringUseCase, logger *zap.Logger) *MonitoringHandler {
	return &MonitoringHandler{
		uc:     uc,
		logger: logger,
	}
}

// ListSessions lists sessions for an exam with optional filters.
//
//	@Summary		List exam sessions
//	@Description	List sessions for an exam. Optional query filters: status and candidateId.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			status			query	string	false	"Session status"
//	@Param			candidateId		query	string	false	"Candidate ID (UUID)"
//	@Success		200				{object}	SessionListResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/exams/{examId}/sessions [get]
func (h *MonitoringHandler) ListSessions(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID"})
		return
	}

	var status *domain.SessionStatus
	if s := c.Query("status"); s != "" {
		st := domain.SessionStatus(s)
		status = &st
	}

	var candidateID *uuid.UUID
	if cidStr := c.Query("candidateId"); cidStr != "" {
		if cid, err := uuid.Parse(cidStr); err == nil {
			candidateID = &cid
		}
	}

	list, err := h.uc.ListSessionsForExam(c.Request.Context(), examID, entID, status, candidateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GetSessionSummary returns a session summary for enterprise monitoring.
//
//	@Summary		Get session summary
//	@Description	Get one session summary by session ID.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200				{object}	SessionResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		404				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/summary [get]
func (h *MonitoringHandler) GetSessionSummary(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	summary, err := h.uc.GetSessionSummary(c.Request.Context(), sessionID, entID)
	if err != nil {
		if err == domain.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

// GetSubmissions lists submissions for an exam.
//
//	@Summary		List exam submissions
//	@Description	List all submissions for an exam under the caller enterprise.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Success		200				{object}	SubmissionListResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/exams/{examId}/submissions [get]
func (h *MonitoringHandler) GetSubmissions(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exam ID"})
		return
	}

	list, err := h.uc.GetSubmissions(c.Request.Context(), examID, entID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GetSubmissionDetail returns a single submission detail by submission ID.
//
//	@Summary		Get submission detail
//	@Description	Get submission detail by submission ID.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			submissionId		path	string	true	"Submission ID (UUID)"
//	@Success		200				{object}	SubmissionResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/submissions/{submissionId} [get]
func (h *MonitoringHandler) GetSubmissionDetail(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	subID, err := uuid.Parse(c.Param("submissionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	sub, err := h.uc.GetSubmissionDetail(c.Request.Context(), subID, entID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

// CandidateGetResult returns candidate-visible result for a submitted session.
//
//	@Summary		Get candidate result
//	@Description	Return candidate result if release policy allows it.
//	@Tags			monitoring
//	@Produce		json
//	@Param			sessionId	path	string	true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string	false	"Candidate ID (fallback if middleware context is absent)"
//	@Success		200			{object}	SubmissionResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		403			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/result [get]
func (h *MonitoringHandler) CandidateGetResult(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing candidate mapping"})
		return
	}

	res, err := h.uc.CandidateGetResult(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		if err == domain.ErrUnauthorizedAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Results not yet released"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}
