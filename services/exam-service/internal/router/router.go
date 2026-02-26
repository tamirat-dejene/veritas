package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/handler"
)

func NewRouter(qh *handler.QuestionHandler, eh *handler.ExamHandler) *gin.Engine {
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	questions := engine.Group("/questions")
	{
		questions.POST("", qh.CreateQuestion)
		questions.GET("", qh.ListQuestions)
	}

	exams := engine.Group("/exams")
	{
		exams.POST("", eh.CreateExam)
		exams.PATCH("/:examId", eh.UpdateExam)
		exams.POST("/:examId/schedule", eh.ScheduleExam)
		exams.POST("/:examId/clone", eh.CloneExam)
	}

	return engine
}
