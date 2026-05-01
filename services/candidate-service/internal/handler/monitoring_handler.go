package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type MonitoringHandler struct {
	uc domain.MonitoringUseCase
}

func NewMonitoringHandler(uc domain.MonitoringUseCase) *MonitoringHandler {
	return &MonitoringHandler{
		uc: uc,
	}
}

// ListSessions lists sessions for an exam with optional filters and pagination.
//
//	@Summary		List exam sessions
//	@Description	List sessions for an exam. Optional query filters: status and candidateId.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			status			query	string	false	"Session status"
//	@Param			candidateId		query	string	false	"Candidate ID (UUID)"
//	@Param			page			query	int		false	"Page number (default 1)"
//	@Param			limit			query	int		false	"Page size (default 10, max 1000)"
//	@Param			sort			query	string	false	"Sort field: created_at|status|started_at"
//	@Param			sort_dir		query	string	false	"Sort direction: asc|desc (default desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.ExamSession]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/sessions [get]
func (h *MonitoringHandler) ListSessions(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
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

	params := pagination.ParseGin(c)
	list, total, err := h.uc.ListSessionsForExam(c.Request.Context(), examID, entID, status, candidateID, params)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pagination.NewPaginatedResponse(list, total, params))
}

// GetSessionSummary returns a session summary for enterprise monitoring.
//
//	@Summary		Get session summary
//	@Description	Get one session summary by session ID.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200				{object}	dto.SessionResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/summary [get]
func (h *MonitoringHandler) GetSessionSummary(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	summary, err := h.uc.GetSessionSummary(c.Request.Context(), sessionID, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SessionResponse{Data: summary})
}

// GetSubmissions lists submissions for an exam with pagination.
//
//	@Summary		List exam submissions
//	@Description	List all submissions for an exam under the caller enterprise with pagination.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			page			query	int		false	"Page number (default 1)"
//	@Param			limit			query	int		false	"Page size (default 10, max 1000)"
//	@Param			sort			query	string	false	"Sort field: created_at|submitted_at"
//	@Param			sort_dir		query	string	false	"Sort direction: asc|desc (default desc)"
//	@Success		200				{object}	pagination.PaginatedResponse[domain.ExamSubmission]
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/submissions [get]
func (h *MonitoringHandler) GetSubmissions(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	examIDParam := c.Param("examId")
	examID, err := uuid.Parse(examIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	params := pagination.ParseGin(c)
	list, total, err := h.uc.GetSubmissions(c.Request.Context(), examID, entID, params)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, pagination.NewPaginatedResponse(list, total, params))
}

// GetSubmissionDetail returns a single submission detail by submission ID.
//
//	@Summary		Get submission detail
//	@Description	Get submission detail by submission ID.
//	@Tags			monitoring
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	false	"Enterprise ID (fallback if middleware context is absent)"
//	@Param			submissionId		path	string	true	"Submission ID (UUID)"
//	@Success		200				{object}	dto.SubmissionResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/submissions/{submissionId} [get]
func (h *MonitoringHandler) GetSubmissionDetail(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	subID, err := uuid.Parse(c.Param("submissionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	sub, err := h.uc.GetSubmissionDetail(c.Request.Context(), subID, entID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SubmissionResponse{Data: sub})
}

// GetCounts returns counts of sessions for an enterprise.
func (h *MonitoringHandler) GetCounts(c *gin.Context) {
	enterpriseIDStr := c.Param("enterpriseId")
	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	count, err := h.uc.GetActiveSessionsCount(c.Request.Context(), enterpriseID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_session_count": count})
}
