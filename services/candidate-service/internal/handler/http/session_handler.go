package http

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/candidate-service/internal/domain"
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

func (h *SessionHandler) ValidateAccess(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.uc.ValidateAccessToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token mapping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *SessionHandler) StartSession(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	session, err := h.uc.StartSession(c.Request.Context(), req.Token, clientIP, userAgent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": session})
}

func (h *SessionHandler) ResumeActive(c *gin.Context) {
	candidateID, err := getCandidateID(c)
	if err != nil {
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

func (h *SessionHandler) GetQuestions(c *gin.Context) {
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

	questions, err := h.uc.GetSessionQuestionsSnapshot(c.Request.Context(), sessionID, candidateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch questions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": questions})
}

func (h *SessionHandler) SaveAnswers(c *gin.Context) {
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

	var req struct {
		QuestionID uuid.UUID       `json:"questionId" binding:"required"`
		AnswerData json.RawMessage `json:"answerData" binding:"required"`
	}
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

func (h *SessionHandler) Submit(c *gin.Context) {
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

	var req struct {
		AutoSubmitted bool `json:"autoSubmitted"`
	}
	_ = c.ShouldBindJSON(&req)

	sub, err := h.uc.SubmitExam(c.Request.Context(), sessionID, candidateID, req.AutoSubmitted)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Exam submitted", "data": sub})
}

func (h *SessionHandler) TerminateWait(c *gin.Context) {
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

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
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

func (h *SessionHandler) ForceExpire(c *gin.Context) {
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

	if err := h.uc.ForceExpireSession(c.Request.Context(), sessionID, entID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session expired"})
}
