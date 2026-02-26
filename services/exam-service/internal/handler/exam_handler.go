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
