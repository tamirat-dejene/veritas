package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/dto"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
	"go.uber.org/zap"
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
//	@Param			body			body	dto.CreateQuestionRequest	true	"Question payload"
//	@Success		201			{object}	sdomain.Question
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
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

	var req dto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	options := make([]sdomain.QuestionOption, len(req.Options))
	for i, o := range req.Options {
		options[i] = sdomain.QuestionOption{
			Content:   o.Content,
			IsCorrect: o.IsCorrect,
		}
	}

	q := sdomain.Question{
		EnterpriseID:   enterpriseID,
		Type:           req.Type,
		Topic:          req.Topic,
		Difficulty:     req.Difficulty,
		Title:          req.Title,
		Content:        req.Content,
		MediaURL:       req.MediaURL,
		Points:             req.Points,
		NegativePoints:     req.NegativePoints,
		ExpectedAnswer:     req.ExpectedAnswer,
		EvaluationCriteria: req.EvaluationCriteria,
		IsActive:           req.IsActive,
		Options:        options,
	}

	created, err := h.usecase.CreateQuestion(c.Request.Context(), &q, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, created)
}

// ListQuestions lists enterprise questions.
//
//	@Summary		List questions
//	@Description	List questions with pagination, sorting and filtering support for the caller enterprise.
//	@Tags			question
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			page			query	int		false	"Page number (default: 1)"
//	@Param			limit			query	int		false	"Number of items per page (default: 10, max: 1000)"
//	@Param			sort			query	string	false	"Sort field (allowed: created_at, updated_at, title, difficulty, type, points) (default: created_at)"
//	@Param			sort_dir		query	string	false	"Sort direction (asc or desc) (default: desc)"
//	@Param			with_correct_answer	query	bool	false	"Include answers and metadata (default: false)"
//	@Success		200			{object}	pagination.PaginatedResponse[sdomain.Question]
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/questions [get]
func (h *QuestionHandler) ListQuestions(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	params := pagination.ParseGin(c)
	withCorrectAnswer := c.Query("with_correct_answer") == "true"

	questions, err := h.usecase.GetQuestions(c.Request.Context(), enterpriseID, params, withCorrectAnswer)
	if err != nil {
		handleError(c, err)
		return
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
//	@Param			with_correct_answer	query	bool	false	"Include answers and metadata (default: false)"
//	@Success		200			{object}	sdomain.Question
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
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

	withCorrectAnswer := c.Query("with_correct_answer") == "true"

	q, err := h.usecase.GetQuestion(c.Request.Context(), questionID, enterpriseID, withCorrectAnswer)
	if err != nil {
		handleError(c, err)
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
//	@Param			body			body	dto.UpdateQuestionRequest	true	"Question payload"
//	@Success		204			{string}	string				"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
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

	var req dto.UpdateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	options := make([]sdomain.QuestionOption, len(req.Options))
	for i, o := range req.Options {
		options[i] = sdomain.QuestionOption{
			Content:   o.Content,
			IsCorrect: o.IsCorrect,
		}
	}

	q := sdomain.Question{
		ID:             questionID,
		EnterpriseID:   enterpriseID,
		Type:           req.Type,
		Topic:          req.Topic,
		Difficulty:     req.Difficulty,
		Title:          req.Title,
		Content:        req.Content,
		MediaURL:       req.MediaURL,
		Points:             req.Points,
		NegativePoints:     req.NegativePoints,
		ExpectedAnswer:     req.ExpectedAnswer,
		EvaluationCriteria: req.EvaluationCriteria,
		IsActive:           req.IsActive,
		Options:        options,
	}

	if err := h.usecase.UpdateQuestion(c.Request.Context(), &q, userID); err != nil {
		handleError(c, err)
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
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
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
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UploadMedia uploads a media file for a question.
//
//	@Summary		Upload media
//	@Description	Upload media (image, video, PDF) for a question. Max size 5MB.
//	@Tags			question
//	@Accept			multipart/form-data
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			questionId		path	string	true	"Question ID (UUID)"
//	@Param			media			formData	file	true	"Media file"
//	@Success		200			{object}	dto.UploadMediaResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		413			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/questions/{questionId}/media [post]
func (h *QuestionHandler) UploadMedia(c *gin.Context) {
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

	// 1. Limit file size (5MB)
	const maxFileSize = 5 * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	file, _, err := c.Request.FormFile("media")
	if err != nil {
		if err.Error() == "http: request body too large" {
			writeError(c, http.StatusRequestEntityTooLarge, "file too large (max 5MB)")
		} else {
			writeError(c, http.StatusBadRequest, "no media file provided")
		}
		return
	}
	defer file.Close()

	// 2. Validate media type
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to read file header")
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to reset file pointer")
		return
	}

	contentType := http.DetectContentType(buff)
	isAllowed := false
	allowedPrefixes := []string{"image/", "video/", "application/pdf"}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		writeError(c, http.StatusBadRequest, "invalid media type (images, videos, PDFs only)")
		return
	}

	// 3. Predictable filename
	mediaFileName := fmt.Sprintf("q_%s", questionID.String())

	url, err := h.usecase.UploadMedia(c.Request.Context(), questionID, enterpriseID, mediaFileName, file)
	if err != nil {
		zap.L().Error("failed to upload question media", zap.Error(err))
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.UploadMediaResponse{MediaURL: url})
}
