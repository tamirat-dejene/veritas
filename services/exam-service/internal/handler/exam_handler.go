package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/dto"
	sdomain "github.com/tamirat-dejene/veritas/shared/domain"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

type ExamHandler struct {
	usecase domain.ExamUsecase
}

func NewExamHandler(uc domain.ExamUsecase) *ExamHandler {
	return &ExamHandler{usecase: uc}
}

// CreateExam creates a new exam.
//
//	@Summary		Create exam
//	@Description	Create one exam under the caller enterprise.
//	@Tags			exam
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string		true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string		true	"Actor user ID (UUID)"
//	@Param			body			body	dto.CreateExamRequest	true	"Exam payload"
//	@Success		201			{object}	sdomain.Exam
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams [post]
func (h *ExamHandler) CreateExam(c *gin.Context) {
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

	var req dto.CreateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	e := sdomain.Exam{
		EnterpriseID:        enterpriseID,
		Title:               req.Title,
		Description:         req.Description,
		DurationMinutes:     req.DurationMinutes,
		PassingScorePercent: req.PassingScorePercent,
		NegativeMarking:     req.NegativeMarking,
		MaxParticipants:     req.MaxParticipants,
		TemplateSourceID:    req.TemplateSourceID,
		Settings:            req.Settings,
	}

	created, err := h.usecase.CreateExam(c.Request.Context(), &e, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, created)
}

// UpdateExam updates a draft exam.
//
//	@Summary		Update exam
//	@Description	Update exam fields by ID.
//	@Tags			exam
//	@Accept			json
//	@Param			X-Enterprise-ID	header	string		true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string		true	"Actor user ID (UUID)"
//	@Param			examId			path	string		true	"Exam ID (UUID)"
//	@Param			body			body	dto.UpdateExamRequest	true	"Exam payload"
//	@Success		204			{string}	string		"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId} [patch]
func (h *ExamHandler) UpdateExam(c *gin.Context) {
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

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	var req dto.UpdateExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	e := sdomain.Exam{
		ID:                  examID,
		EnterpriseID:        enterpriseID,
		Title:               req.Title,
		Description:         req.Description,
		DurationMinutes:     int(req.DurationMinutes),
		PassingScorePercent: req.PassingScorePercent,
		NegativeMarking:     req.NegativeMarking,
		MaxParticipants:     req.MaxParticipants,
		Settings:            req.Settings,
	}

	if err := h.usecase.UpdateExam(c.Request.Context(), &e, userID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ScheduleExam schedules an exam window.
//
//	@Summary		Schedule exam
//	@Description	Set start and end times for an exam.
//	@Tags			exam
//	@Accept			json
//	@Param			X-Enterprise-ID	header	string				true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string				true	"Actor user ID (UUID)"
//	@Param			examId			path	string				true	"Exam ID (UUID)"
//	@Param			body			body	dto.ScheduleExamRequest	true	"Schedule payload"
//	@Success		204			{string}	string				"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/schedule [post]
func (h *ExamHandler) ScheduleExam(c *gin.Context) {
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

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	var req dto.ScheduleExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid startTime format")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid endTime format")
		return
	}

	if !startTime.Before(endTime) {
		writeError(c, http.StatusBadRequest, "start time must be before end time")
		return
	}

	if err := h.usecase.ScheduleExam(c.Request.Context(), examID, enterpriseID, startTime, endTime, userID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CloneExam clones an existing exam into a new draft.
//
//	@Summary		Clone exam
//	@Description	Clone exam content into a new exam with provided title.
//	@Tags			exam
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string			true	"Enterprise ID (UUID)"
//	@Param			X-User-ID	header	string			true	"Actor user ID (UUID)"
//	@Param			examId			path	string			true	"Source Exam ID (UUID)"
//	@Param			body			body	dto.CloneExamRequest	true	"Clone payload"
//	@Success		201			{object}	sdomain.Exam
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/clone [post]
func (h *ExamHandler) CloneExam(c *gin.Context) {
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

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	var req dto.CloneExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	cloned, err := h.usecase.CloneExam(c.Request.Context(), examID, enterpriseID, req.Title, userID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, cloned)
}

// ListExams lists enterprise exams.
//
//	@Summary		List exams
//	@Description	List exams with pagination, sorting and filtering support for the caller enterprise.
//	@Tags			exam
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			page			query	int		false	"Page number (default: 1)"
//	@Param			limit			query	int		false	"Number of items per page (default: 10, max: 1000)"
//	@Param			sort			query	string	false	"Sort field (allowed: created_at, updated_at, title, duration_minutes, passing_score_percent, status) (default: created_at)"
//	@Param			sort_dir		query	string	false	"Sort direction (asc or desc) (default: desc)"
//	@Success		200			{object}	pagination.PaginatedResponse[sdomain.Exam]
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams [get]
func (h *ExamHandler) ListExams(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	params := pagination.ParseGin(c)

	exams, err := h.usecase.GetExams(c.Request.Context(), enterpriseID, params)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, exams)
}

// GetExam gets one exam by ID.
//
//	@Summary		Get exam
//	@Description	Get one exam for the caller enterprise.
//	@Tags			exam
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Success		200			{object}	sdomain.Exam
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId} [get]
func (h *ExamHandler) GetExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	e, err := h.usecase.GetExam(c.Request.Context(), examID, enterpriseID)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, e)
}

// GetExamQuestions lists mapped questions for an exam.
//
//	@Summary		Get exam questions
//	@Description	Get question mappings for one exam with pagination.
//	@Tags			exam
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			page			query	int		false	"Page number (default: 1)"
//	@Param			limit			query	int		false	"Number of items per page (default: 10, max: 1000)"
//	@Param			sort			query	string	false	"Sort field (allowed: order_index, points_override) (default: order_index)"
//	@Param			sort_dir		query	string	false	"Sort direction (asc or desc) (default: desc)"
//	@Param			with_correct_answer	query	bool	false	"Include answers and metadata (default: false)"
//	@Success		200			{object}	pagination.PaginatedResponse[sdomain.ExamQuestion]
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/questions [get]
func (h *ExamHandler) GetExamQuestions(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	params := pagination.ParseGin(c)
	withCorrectAnswer := c.Query("with_correct_answer") == "true"

	questions, err := h.usecase.GetExamQuestions(c.Request.Context(), examID, enterpriseID, params, withCorrectAnswer)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, questions)
}

// PublishExam publishes a draft/scheduled exam.
//
//	@Summary		Publish exam
//	@Description	Publish exam after validation checks.
//	@Tags			exam
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/publish [post]
func (h *ExamHandler) PublishExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	if err := h.usecase.PublishExam(c.Request.Context(), examID, enterpriseID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CloseExam closes an active exam.
//
//	@Summary		Close exam
//	@Description	Close an active exam.
//	@Tags			exam
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		409			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/close [post]
func (h *ExamHandler) CloseExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	if err := h.usecase.CloseExam(c.Request.Context(), examID, enterpriseID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// DeleteExam deletes an exam.
//
//	@Summary		Delete exam
//	@Description	Delete one exam by ID.
//	@Tags			exam
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId} [delete]
func (h *ExamHandler) DeleteExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	if err := h.usecase.DeleteExam(c.Request.Context(), examID, enterpriseID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AddQuestionsToExam maps multiple questions into an exam.
//
//	@Summary		Add questions to exam
//	@Description	Attach multiple questions to an exam with optional override points and order indices.
//	@Tags			exam
//	@Accept			json
//	@Produce		json
//	@Param			X-Enterprise-ID	header	string					true	"Enterprise ID (UUID)"
//	@Param			examId			path	string					true	"Exam ID (UUID)"
//	@Param			body			body	dto.AddExamQuestionsBulkRequest	true	"Exam questions payload"
//	@Success		201			{array}	    sdomain.ExamQuestion
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/questions [post]
func (h *ExamHandler) AddQuestionsToExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	var req dto.AddExamQuestionsBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	var inputs []sdomain.ExamQuestionInput
	for _, qReq := range req.Questions {
		qID, err := uuid.Parse(qReq.QuestionID)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Sprintf("invalid question ID: %s", qReq.QuestionID))
			return
		}
		inputs = append(inputs, sdomain.ExamQuestionInput{
			QuestionID:     qID,
			PointsOverride: qReq.PointsOverride,
			OrderIndex:     qReq.OrderIndex,
		})
	}

	eqs, err := h.usecase.AddQuestionsToExam(c.Request.Context(), enterpriseID, examID, inputs)
	if err != nil {
		handleError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, eqs)
}

// RemoveQuestionFromExam removes a mapped question from an exam.
//
//	@Summary		Remove question from exam
//	@Description	Remove one question mapping from exam.
//	@Tags			exam
//	@Param			X-Enterprise-ID	header	string	true	"Enterprise ID (UUID)"
//	@Param			examId			path	string	true	"Exam ID (UUID)"
//	@Param			questionId		path	string	true	"Question ID (UUID)"
//	@Success		204			{string}	string	"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/questions/{questionId} [delete]
func (h *ExamHandler) RemoveQuestionFromExam(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	questionIDStr := c.Param("questionId")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid question ID")
		return
	}

	if err := h.usecase.RemoveQuestionFromExam(c.Request.Context(), enterpriseID, examID, questionID); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateExamQuestion updates exam-question mapping details.
//
//	@Summary		Update exam question mapping
//	@Description	Update points override or order index for an exam question mapping.
//	@Tags			exam
//	@Accept			json
//	@Param			X-Enterprise-ID	header	string					true	"Enterprise ID (UUID)"
//	@Param			examId			path	string					true	"Exam ID (UUID)"
//	@Param			questionId		path	string					true	"Question ID (UUID)"
//	@Param			body			body	dto.UpdateExamQuestionRequest	true	"Update mapping payload"
//	@Success		204			{string}	string					"No Content"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/exams/{examId}/questions/{questionId} [patch]
func (h *ExamHandler) UpdateExamQuestion(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	examIDStr := c.Param("examId")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid exam ID")
		return
	}

	questionIDStr := c.Param("questionId")
	questionID, err := uuid.Parse(questionIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid question ID")
		return
	}

	var req dto.UpdateExamQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.usecase.UpdateExamQuestion(c.Request.Context(), enterpriseID, examID, questionID, req.PointsOverride, req.OrderIndex); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetCounts returns counts of exams for an enterprise.
func (h *ExamHandler) GetCounts(c *gin.Context) {
	enterpriseIDStr := c.Param("enterpriseId")
	enterpriseID, err := uuid.Parse(enterpriseIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid enterprise ID")
		return
	}

	count, err := h.usecase.GetActiveExamsCount(c.Request.Context(), enterpriseID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"active_exam_count": count})
}


