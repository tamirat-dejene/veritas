package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tamirat-dejene/veritas/services/exam-service/internal/handler"
)

func NewRouter(qh *handler.QuestionHandler, eh *handler.ExamHandler) *gin.Engine {
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	engine.GET("/health", healthCheck)

	questions := engine.Group("/questions")
	{
		questions.POST("", qh.CreateQuestion)
		questions.GET("", qh.ListQuestions)
		questions.GET("/:questionId", qh.GetQuestion)
		questions.PATCH("/:questionId", qh.UpdateQuestion)
		questions.DELETE("/:questionId", qh.DeleteQuestion)
	}

	exams := engine.Group("/exams")
	{
		exams.POST("", eh.CreateExam)
		exams.GET("", eh.ListExams)
		exams.GET("/:examId", eh.GetExam)
		exams.GET("/:examId/questions", eh.GetExamQuestions)
		exams.PATCH("/:examId", eh.UpdateExam)
		exams.POST("/:examId/schedule", eh.ScheduleExam)
		exams.POST("/:examId/clone", eh.CloneExam)
		exams.POST("/:examId/publish", eh.PublishExam)
		exams.POST("/:examId/close", eh.CloseExam)
		exams.DELETE("/:examId", eh.DeleteExam)

		// Exam Question Assembly
		exams.POST("/:examId/questions", eh.AddQuestionToExam)
		exams.DELETE("/:examId/questions/:questionId", eh.RemoveQuestionFromExam)
		exams.PATCH("/:examId/questions/:questionId", eh.UpdateExamQuestion)

		// Exam Randomization Rules
		exams.POST("/:examId/rules", eh.AddRandomizationRule)
		exams.PATCH("/:examId/rules/:ruleId", eh.UpdateRandomizationRule)
		exams.DELETE("/:examId/rules/:ruleId", eh.DeleteRandomizationRule)
	}

	// Swagger UI — available at /swagger/index.html
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return engine
}

// healthCheck returns 200 OK if the service is running.
//
//	@Summary		Health check
//	@ID			healthCheck
//	@Description	Returns a simple JSON indicating the service is alive.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Service is healthy"
//	@Router			/health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "exam-service"})
}
