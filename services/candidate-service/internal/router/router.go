package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	c_http "github.com/tamirat-dejene/veritas/services/candidate-service/internal/handler"
)

func NewRouter(
	ch *c_http.CandidateHandler,
	eh *c_http.EnrollmentHandler,
	sh *c_http.SessionHandler,
	mh *c_http.MonitoringHandler,
) *gin.Engine {
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	engine.GET("/health", healthCheck)

	// Role checks and enterprise_id mapping happen upstream (API Gateway) or via intercept middlewares.

	candidates := engine.Group("/candidates")
	{
		candidates.POST("", ch.Create)
		candidates.POST("/bulk", ch.BulkUpload)
		candidates.GET("", ch.List)
		candidates.GET("/:candidateId", ch.Get)
		candidates.PATCH("/:candidateId", ch.Update)
		candidates.PATCH("/:candidateId/deactivate", ch.Deactivate)
	}

	// Enrollment rules usually hang off exams logically
	exams := engine.Group("/exams")
	{
		exams.POST("/:examId/enrollments", eh.Enroll)
		exams.GET("/:examId/enrollments", eh.ListByExam)
		exams.GET("/:examId/sessions", mh.ListSessions)
		exams.GET("/:examId/submissions", mh.GetSubmissions)
	}

	enrollments := engine.Group("/enrollments")
	{
		enrollments.GET("/:enrollmentId", eh.Get)
		enrollments.POST("/:enrollmentId/regenerate-token", eh.RegenerateToken)
		enrollments.PATCH("/:enrollmentId/revoke", eh.Revoke)
		enrollments.POST("/:enrollmentId/reset-attempts", eh.ResetAttempts)
	}

	access := engine.Group("/access")
	{
		access.POST("/validate", sh.ValidateAccess)
	}

	sessions := engine.Group("/sessions")
	{
		sessions.POST("/start", sh.StartSession)
		sessions.GET("/me/active", sh.ResumeActive)
		sessions.GET("/:sessionId", sh.GetDetails)

		sessions.GET("/:sessionId/questions", sh.GetQuestions)
		sessions.PATCH("/:sessionId/answers", sh.SaveAnswers)
		sessions.GET("/:sessionId/answers", sh.GetMyAnswers)

		sessions.POST("/:sessionId/submit", sh.Submit)
		sessions.POST("/:sessionId/terminate", sh.TerminateWait)
		sessions.POST("/:sessionId/expire", sh.ForceExpire)

		sessions.GET("/:sessionId/summary", mh.GetSessionSummary)
		sessions.GET("/:sessionId/result", mh.CandidateGetResult)
	}

	submissions := engine.Group("/submissions")
	{
		submissions.GET("/:submissionId", mh.GetSubmissionDetail)
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
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "candidate-service"})
}
