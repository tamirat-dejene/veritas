package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

type SessionHandler struct {
	uc     domain.SessionUseCase
	logger *zap.Logger
}

func NewSessionHandler(uc domain.SessionUseCase, logger *zap.Logger) *SessionHandler {
	return &SessionHandler{
		uc:     uc,
		logger: logger,
	}
}

// Ensure candidate ID inside token payload
func getCandidateID(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("subject_id")
	if !exists {
		// fallback
		headerVal := c.GetHeader("X-Subject-Id")
		if headerVal != "" {
			return uuid.Parse(headerVal)
		}
		return uuid.Nil, domain.ErrUnauthorizedAccess
	}
	return uuid.Parse(val.(string))
}

// ValidateAccess validates a one-time access token and returns mapped metadata.
//
//	@Summary		Validate access token
//	@Description	Validate exam access token before session start.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			body	body		AccessValidateRequest	true	"Access token"
//	@Success		200		{object}	AccessValidateResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/access/validate [post]
func (h *SessionHandler) ValidateAccess(c *gin.Context) {
	var req AccessValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.uc.ValidateAccessToken(c.Request.Context(), req.Token)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Token validation failed", zap.Error(err), zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

// StartSession starts a new exam session for a validated token.
//
//	@Summary		Start session
//	@Description	Create and initialize a candidate exam session.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SessionStartRequest	true	"Session start payload"
//	@Success		201		{object}	SessionResponse
//	@Failure		400		{object}	ErrorResponse
//	@Router			/sessions/start [post]
func (h *SessionHandler) StartSession(c *gin.Context) {
	var req SessionStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	session, err := h.uc.StartSession(c.Request.Context(), req.Token, clientIP, userAgent)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Failed to start session", zap.Error(err), zap.String("ip", clientIP))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": session})
}

// ResumeActive returns the caller candidate's currently active session.
//
//	@Summary		Resume active session
//	@Description	Return the active session for the authenticated candidate.
//	@Tags			session
//	@Produce		json
//	@Param			X-Subject-Id	header	string	false	"Candidate ID (fallback if middleware context is absent)"
//	@Success		200			{object}	SessionResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/me/active [get]
func (h *SessionHandler) ResumeActive(c *gin.Context) {
	candidateID, err := getCandidateID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Candidate mapping missing in request", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Candidate mapping missing"})
		return
	}

	session, err := h.uc.ResumeActiveSession(c.Request.Context(), candidateID)
	if err != nil {
		if err == domain.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active session found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": session})
}

// GetDetails retrieves full session details.
//
//	@Summary		Get session details
//	@Description	Get session details for a candidate/admin context.
//	@Tags			session
//	@Produce		json
//	@Param			sessionId	path	string	true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string	false	"Candidate ID (fallback if middleware context is absent)"
//	@Success		200			{object}	SessionResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId} [get]
func (h *SessionHandler) GetDetails(c *gin.Context) {
	// Either candidate or Admin/Staff
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	subID, _ := getCandidateID(c)

	session, err := h.uc.GetSessionDetails(c.Request.Context(), sessionID, subID, "role_placeholder")
	if err != nil {
		if err == domain.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": session})
}

// GetQuestions returns the frozen question snapshot for a session.
//
//	@Summary		Get session questions
//	@Description	Get question snapshots for a candidate session.
//	@Tags			session
//	@Produce		json
//	@Param			sessionId	path	string	true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string	false	"Candidate ID (fallback if middleware context is absent)"
//	@Success		200			{object}	SessionQuestionListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/questions [get]
func (h *SessionHandler) GetQuestions(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Unauthorized context in GetQuestions", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized context"})
		return
	}

	questions, err := h.uc.GetSessionQuestionsSnapshot(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch questions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": questions})
}

// SaveAnswers upserts one answer for a session question.
//
//	@Summary		Save session answer
//	@Description	Save or update one question answer in a session.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			sessionId	path	string				true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string				false	"Candidate ID (fallback if middleware context is absent)"
//	@Param			body			body	SaveAnswerRequest	true	"Answer payload"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/answers [patch]
func (h *SessionHandler) SaveAnswers(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Unauthorized context in SaveAnswers", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized context"})
		return
	}

	var req SaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.uc.SaveAnswers(c.Request.Context(), sessionID, candidateID, req.QuestionID, req.AnswerData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Answers saved"})
}

// GetMyAnswers returns the caller candidate's saved answers for the session.
//
//	@Summary		Get my answers
//	@Description	Return answers saved by the authenticated candidate for a session.
//	@Tags			session
//	@Produce		json
//	@Param			sessionId	path	string	true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string	false	"Candidate ID (fallback if middleware context is absent)"
//	@Success		200			{object}	SessionAnswerListResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/answers [get]
func (h *SessionHandler) GetMyAnswers(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized context"})
		return
	}

	ans, err := h.uc.GetMyAnswers(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ans})
}

// Submit submits a session for grading.
//
//	@Summary		Submit exam session
//	@Description	Submit candidate exam session and create a submission record.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			sessionId	path	string			true	"Session ID (UUID)"
//	@Param			X-Subject-Id	header	string			false	"Candidate ID (fallback if middleware context is absent)"
//	@Param			body			body	SubmitRequest	false	"Submission metadata"
//	@Success		201			{object}	SubmitResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/submit [post]
func (h *SessionHandler) Submit(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	candidateID, err := getCandidateID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Unauthorized context in Submit", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized context"})
		return
	}

	var req SubmitRequest
	_ = c.ShouldBindJSON(&req)

	sub, err := h.uc.SubmitExam(c.Request.Context(), sessionID, candidateID, req.AutoSubmitted)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Error("Exam submission failed", zap.Error(err), zap.String("sessionID", sessionID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Exam submitted", "data": sub})
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
//	@Param			body				body		TerminateSessionRequest	true	"Termination reason"
//	@Success		200				{object}	MessageResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/terminate [post]
func (h *SessionHandler) TerminateWait(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Enterprise ID missing in request", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	var req TerminateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.uc.TerminateSession(c.Request.Context(), sessionID, entID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session terminated"})
}

// ForceExpire forcefully marks a session as expired.
//
//	@Summary		Expire session
//	@Description	Force-expire a session by enterprise/admin action.
//	@Tags			session
//	@Produce		json
//	@Param			X-Enterprise-Id	header	string	true	"Enterprise ID"
//	@Param			sessionId		path	string	true	"Session ID (UUID)"
//	@Success		200				{object}	MessageResponse
//	@Failure		400				{object}	ErrorResponse
//	@Failure		401				{object}	ErrorResponse
//	@Failure		500				{object}	ErrorResponse
//	@Router			/sessions/{sessionId}/expire [post]
func (h *SessionHandler) ForceExpire(c *gin.Context) {
	entID, err := getEnterpriseID(c)
	if err != nil {
		logger.WithContext(c.Request.Context(), h.logger).Warn("Enterprise ID missing in request", zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Enterprise ID missing"})
		return
	}

	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	if err := h.uc.ForceExpireSession(c.Request.Context(), sessionID, entID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session expired"})
}
