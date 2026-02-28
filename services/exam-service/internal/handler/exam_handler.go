package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/domain"
)

type ExamHandler struct {
	usecase domain.ExamUsecase
}

func NewExamHandler(uc domain.ExamUsecase) *ExamHandler {
	return &ExamHandler{usecase: uc}
}

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

	var e domain.Exam
	if err := c.ShouldBindJSON(&e); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	e.EnterpriseID = enterpriseID

	created, err := h.usecase.CreateExam(c.Request.Context(), &e, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to create exam")
		return
	}

	writeJSON(c, http.StatusCreated, created)
}

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

	var e domain.Exam
	if err := c.ShouldBindJSON(&e); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	e.ID = examID
	e.EnterpriseID = enterpriseID

	if err := h.usecase.UpdateExam(c.Request.Context(), &e, userID); err != nil {
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		if err == domain.ErrInvalidStatus {
			writeError(c, http.StatusConflict, "cannot update non-draft exam")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to update exam")
		return
	}

	c.Status(http.StatusNoContent)
}

type scheduleRequest struct {
	StartTime time.Time `json:"startTime" binding:"required"`
	EndTime   time.Time `json:"endTime" binding:"required"`
}

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

	var req scheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.usecase.ScheduleExam(c.Request.Context(), examID, enterpriseID, req.StartTime, req.EndTime, userID); err != nil {
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		if err == domain.ErrInvalidStatus {
			writeError(c, http.StatusConflict, "exam status must be draft or scheduled")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to schedule exam")
		return
	}

	c.Status(http.StatusNoContent)
}

type cloneRequest struct {
	Title string `json:"title" binding:"required"`
}

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

	var req cloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	cloned, err := h.usecase.CloneExam(c.Request.Context(), examID, enterpriseID, req.Title, userID)
	if err != nil {
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "source exam not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to clone exam")
		return
	}

	writeJSON(c, http.StatusCreated, cloned)
}

func (h *ExamHandler) ListExams(c *gin.Context) {
	enterpriseID, ok := getEnterpriseID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing enterprise ID")
		return
	}

	exams, err := h.usecase.GetExams(c.Request.Context(), enterpriseID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "failed to fetch exams")
		return
	}

	if exams == nil {
		exams = make([]*domain.Exam, 0)
	}

	writeJSON(c, http.StatusOK, exams)
}

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
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to fetch exam")
		return
	}

	writeJSON(c, http.StatusOK, e)
}

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

	questions, err := h.usecase.GetExamQuestions(c.Request.Context(), examID, enterpriseID)
	if err != nil {
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to fetch exam questions")
		return
	}

	if questions == nil {
		questions = make([]*domain.ExamQuestion, 0)
	}

	writeJSON(c, http.StatusOK, questions)
}

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
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		if err == domain.ErrInvalidStatus {
			writeError(c, http.StatusConflict, "exam status must be draft or scheduled")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

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
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		if err == domain.ErrInvalidStatus {
			writeError(c, http.StatusConflict, "exam must be active to close")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to close exam")
		return
	}

	c.Status(http.StatusNoContent)
}

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
		if err == domain.ErrExamNotFound {
			writeError(c, http.StatusNotFound, "exam not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "failed to delete exam")
		return
	}

	c.Status(http.StatusNoContent)
}

type examQuestionRequest struct {
	QuestionID     uuid.UUID `json:"questionId" binding:"required"`
	PointsOverride *int      `json:"pointsOverride,omitempty"`
	OrderIndex     *int      `json:"orderIndex,omitempty"`
}

func (h *ExamHandler) AddQuestionToExam(c *gin.Context) {
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

	var req examQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	eq, err := h.usecase.AddQuestionToExam(c.Request.Context(), enterpriseID, examID, req.QuestionID, req.PointsOverride, req.OrderIndex)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusCreated, eq)
}

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
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

type updateExamQuestionRequest struct {
	PointsOverride *int `json:"pointsOverride,omitempty"`
	OrderIndex     *int `json:"orderIndex,omitempty"`
}

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

	var req updateExamQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.usecase.UpdateExamQuestion(c.Request.Context(), enterpriseID, examID, questionID, req.PointsOverride, req.OrderIndex); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

type examRuleRequest struct {
	Topic         *string                 `json:"topic,omitempty"`
	Difficulty    *domain.DifficultyLevel `json:"difficulty,omitempty"`
	QuestionCount int                     `json:"questionCount" binding:"required"`
}

func (h *ExamHandler) AddRandomizationRule(c *gin.Context) {
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

	var req examRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	rule, err := h.usecase.AddRandomizationRule(c.Request.Context(), enterpriseID, examID, req.Topic, req.Difficulty, req.QuestionCount)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(c, http.StatusCreated, rule)
}

func (h *ExamHandler) UpdateRandomizationRule(c *gin.Context) {
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

	ruleIDStr := c.Param("ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	var req examRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.usecase.UpdateRandomizationRule(c.Request.Context(), enterpriseID, examID, ruleID, req.Topic, req.Difficulty, req.QuestionCount); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ExamHandler) DeleteRandomizationRule(c *gin.Context) {
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

	ruleIDStr := c.Param("ruleId")
	ruleID, err := uuid.Parse(ruleIDStr)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid rule ID")
		return
	}

	if err := h.usecase.DeleteRandomizationRule(c.Request.Context(), enterpriseID, examID, ruleID); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
