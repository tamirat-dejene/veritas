package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type QuestionHandler struct {
	usecase domain.QuestionUsecase
}

func NewQuestionHandler(uc domain.QuestionUsecase) *QuestionHandler {
	return &QuestionHandler{usecase: uc}
}

func (h *QuestionHandler) CreateQuestion(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing user ID")
		return
	}

	var q domain.Question
	if err := c.ShouldBindJSON(&q); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	q.EnterpriseID = enterpriseID

	created, err := h.usecase.CreateQuestion(c.Request.Context(), &q, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create question")
		return
	}

	writeJSON(c, http.StatusCreated, created)
}

func (h *QuestionHandler) ListQuestions(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	questions, err := h.usecase.GetQuestions(c.Request.Context(), enterpriseID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to fetch questions")
		return
	}

	// Make sure we never return nil JSON arrays if questions is empty
	if questions == nil {
		questions = make([]*domain.Question, 0)
	}

	writeJSON(c, http.StatusOK, questions)
}
