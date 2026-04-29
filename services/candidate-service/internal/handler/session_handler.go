package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/dto"
)

type SessionHandler struct {
	uc domain.SessionUseCase
}

func NewSessionHandler(uc domain.SessionUseCase) *SessionHandler {
	return &SessionHandler{
		uc: uc,
	}
}

// ValidateAccess validates a one-time access token and returns mapped metadata.
//
//	@Summary		Validate access token
//	@Description	Validate exam access token before session start.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			X-Enrollment-Id	header		string						true	"Enrollment ID"
//	@Param			X-Enterprise-Id	header		string						true	"Enterprise ID"
//	@Success		200				{object}	dto.AccessValidateResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Router			/access/validate [post]
func (h *SessionHandler) ValidateAccess(c *gin.Context) {
	eid, err := getEnrollmentID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnrollmentIDMissing.Error()})
		return
	}
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	res, err := h.uc.ValidateAccessToken(c.Request.Context(), eid, entID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrInvalidToken.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AccessValidateResponse{Data: res})
}

// StartSession starts a new exam session for a validated token.
//
//	@Summary		Start session
//	@Description	Create and initialize a candidate exam session.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			X-Enrollment-Id	header		string					true	"Enrollment ID"
//	@Param			X-Enterprise-Id	header		string					true	"Enterprise ID"
//	@Success		201				{object}	dto.SessionResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Router			/sessions/start [post]
func (h *SessionHandler) StartSession(c *gin.Context) {
	eid, err := getEnrollmentID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnrollmentIDMissing.Error()})
		return
	}
	entID, err := getEnterpriseID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrEnterpriseIDMissing.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	session, err := h.uc.StartSession(c.Request.Context(), eid, entID, clientIP, userAgent)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SessionResponse{Data: session})
}

// ResumeActive returns the caller candidate's currently active session.
//
//	@Summary		Resume active session
//	@Description	Return the active session for the authenticated candidate.
//	@Tags			session
//	@Produce		json
//	@Param			X-Subject-Id	header	string	true	"Candidate ID"
//	@Success		200				{object}	dto.SessionResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		404				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/sessions/me/active [get]
func (h *SessionHandler) ResumeActive(c *gin.Context) {
	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrCandidateIDMissing.Error()})
		return
	}

	session, err := h.uc.ResumeActiveSession(c.Request.Context(), candidateID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SessionResponse{Data: session})
}

// GetDetails retrieves full session details.
//
//	@Summary		Get session details
//	@Description	Get session details for a candidate/admin context.
//	@Tags			session
//	@Produce		json
//	@Param			X-Subject-Id	header	string	true	"Candidate ID"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200			{object}	dto.SessionResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId} [get]
func (h *SessionHandler) GetDetails(c *gin.Context) {
	// Either candidate or Admin/Staff
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	subID, _ := getCandidateID(c)

	session, err := h.uc.GetSessionDetails(c.Request.Context(), sessionID, subID, "role_placeholder")
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SessionResponse{Data: session})
}

// GetQuestions returns the frozen question snapshot for a session.
//
//	@Summary		Get session questions
//	@Description	Get question snapshots for a candidate session.
//	@Tags			session
//	@Produce		json
//	@Param			X-Subject-Id	header	string	true	"Candidate ID"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200			{object}	dto.SessionQuestionListResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/questions [get]
func (h *SessionHandler) GetQuestions(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrUnauthorizedContext.Error()})
		return
	}

	questions, err := h.uc.GetSessionQuestionsSnapshot(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SessionQuestionListResponse{Data: questions})
}

// SaveAnswers upserts one answer for a session question.
//
//	@Summary		Save session answer
//	@Description	Save or update one question answer in a session. One of the two fields in the answer data must be non null. Both fields being non null or null is not allowed.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			X-Subject-Id	header	string					true	"Candidate ID"
//	@Param			sessionId		path	string					true	"Session ID (UUID)"
//	@Param			body			body	dto.SaveAnswerRequestSwag	true	"Answer payload"
//	@Success		200			{object}	dto.MessageResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/answers [patch]
func (h *SessionHandler) SaveAnswers(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrUnauthorizedContext.Error()})
		return
	}

	var req dto.SaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	err = h.uc.SaveAnswers(c.Request.Context(), sessionID, candidateID, req.SessionQuestionID, req.AnswerData)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Answers saved"})
}

// GetMyAnswers returns the caller candidate's saved answers for the session.
//
//	@Summary		Get my answers
//	@Description	Return answers saved by the authenticated candidate for a session.
//	@Tags			session
//	@Produce		json
//	@Param			X-Subject-Id	header	string	true	"Candidate ID"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200			{object}	dto.SessionAnswerListResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/answers [get]
func (h *SessionHandler) GetMyAnswers(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrUnauthorizedContext.Error()})
		return
	}

	ans, err := h.uc.GetMyAnswers(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SessionAnswerListResponse{Data: ans})
}

// Submit submits a session for grading.
//
//	@Summary		Submit exam session
//	@Description	Submit candidate exam session and create a submission record.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			X-Subject-Id	header	string			true	"Candidate ID"
//	@Param			sessionId		path	string			true	"Session ID (UUID)"
//	@Param			body			body	dto.SubmitRequest	false	"Submission metadata"
//	@Success		201			{object}	dto.SubmitResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/submit [post]
func (h *SessionHandler) Submit(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: domain.ErrInvalidIDFormat.Error()})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: domain.ErrUnauthorizedContext.Error()})
		return
	}

	var req dto.SubmitRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	sub, err := h.uc.SubmitExam(c.Request.Context(), sessionID, candidateID, req.AutoSubmitted)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SubmitResponse{Message: "Exam submitted", Data: sub})
}

// TerminateWait terminates an active session with a reason.
//
//	@Summary		Terminate session
//	@Description	Terminate a session on enterprise/admin action.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-Id	header		string				true	"Enterprise ID"
//	@Param			sessionId		path		string				true	"Session ID (UUID)"
//	@Param			body				body		dto.TerminateSessionRequest	true	"Termination reason"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/terminate [post]
func (h *SessionHandler) TerminateWait(c *gin.Context) {
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

	var req dto.TerminateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.uc.TerminateSession(c.Request.Context(), sessionID, entID, req.Reason); err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Session terminated"})
}

// ForceExpire forcefully marks a session as expired.
//
//	@Summary		Expire session
//	@Description	Force-expire a session by enterprise/admin action.
//	@Tags			session
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	true	"Enterprise ID"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200				{object}	dto.MessageResponse
//	@Failure		400				{object}	dto.ErrorResponse
//	@Failure		401				{object}	dto.ErrorResponse
//	@Failure		500				{object}	dto.ErrorResponse
//	@Router			/sessions/{sessionId}/expire [post]
func (h *SessionHandler) ForceExpire(c *gin.Context) {
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

	if err := h.uc.ForceExpireSession(c.Request.Context(), sessionID, entID); err != nil {
		HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Session expired"})
}
