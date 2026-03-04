package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type QuestionHandler struct {
	usecase domain.QuestionUsecase
}

func NewQuestionHandler(uc domain.QuestionUsecase) *QuestionHandler {
	return &QuestionHandler{usecase: uc}
}

// CreateQuestion creates a new question in enterprise question bank.
//
//	@Summary		Create question
//	@Description	Create one question under the caller enterprise.
//	@Tags			question
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string				true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string				true	"Actor user ID (UUID)"
//	@Param			body			body	domain.Question	true	"Question payload"
//	@Success		201			{object}	domain.Question
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/questions [post]
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

// ListQuestions lists enterprise questions.
//
//	@Summary		List questions
//	@Description	List all questions for the caller enterprise.
//	@Tags			question
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Success		200			{array}	domain.Question
//	@Failure		401			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/questions [get]
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

// GetQuestion gets one question by ID.
//
//	@Summary		Get question
//	@Description	Get a question by ID for the caller enterprise.
//	@Tags			question
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			questionId		path	string	true	"Question ID (UUID)"
//	@Success		200			{object}	domain.Question
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/questions/{questionId} [get]
func (h *QuestionHandler) GetQuestion(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	questionIDStr := c.Param("questionId")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid question ID")
		return
	}

	q, err := h.usecase.GetQuestion(c.Request.Context(), questionID, enterpriseID)
	if err != nil {
		if err == domain.ErrQuestionNotFound {
			writeError(c, http.StatusNotFound, "question not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to fetch question")
		return
	}

	writeJSON(c, http.StatusOK, q)
}

// UpdateQuestion updates an existing question.
//
//	@Summary		Update question
//	@Description	Update question fields by ID.
//	@Tags			question
//	@Accept			json
//	@Param			X-Enterprise-ID	header	string				true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string				true	"Actor user ID (UUID)"
//	@Param			questionId		path	string				true	"Question ID (UUID)"
//	@Param			body			body	domain.Question	true	"Question payload"
//	@Success		204			{string}	string				"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/questions/{questionId} [patch]
func (h *QuestionHandler) UpdateQuestion(c *gin.Context) {
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

	questionIDStr := c.Param("questionId")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid question ID")
		return
	}

	var q domain.Question
	if err := c.ShouldBindJSON(&q); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	q.ID = questionID
	q.EnterpriseID = enterpriseID

	if err := h.usecase.UpdateQuestion(c.Request.Context(), &q, userID); err != nil {
		if err == domain.ErrQuestionNotFound {
			writeError(c, http.StatusNotFound, "question not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to update question")
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteQuestion deletes a question by ID.
//
//	@Summary		Delete question
//	@Description	Delete one question from the caller enterprise.
//	@Tags			question
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			questionId		path	string	true	"Question ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/questions/{questionId} [delete]
func (h *QuestionHandler) DeleteQuestion(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	questionIDStr := c.Param("questionId")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid question ID")
		return
	}

	if err := h.usecase.DeleteQuestion(c.Request.Context(), questionID, enterpriseID); err != nil {
		if err == domain.ErrQuestionNotFound {
			writeError(c, http.StatusNotFound, "question not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to delete question")
		return
	}

	c.Status(http.StatusNoContent)
}
