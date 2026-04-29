package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	c_http "github.com/tamirat-dejene/veritas/services/candidate-service/internal/handler"
	smw "github.com/tamirat-dejene/veritas/shared/pkg/middleware"
)

func NewRouter(
	ch *c_http.CandidateHandler,
	eh *c_http.EnrollmentHandler,
	sh *c_http.SessionHandler,
	mh *c_http.MonitoringHandler,
) *gin.Engine {
	engine := gin.New()
	engine.Use(
		smw.Recovery(),
		smw.RequestID(),
		smw.Logging(),
	)

	engine.GET("/health", healthCheck)

	// ── Candidate profile management ──────────────────────────────────────────
	candidates := engine.Group("/candidates")
	{
		candidates.POST("", ch.Create)
		candidates.POST("/bulk", ch.BulkUpload)
		candidates.GET("", ch.List)
		candidates.GET("/:candidateId", ch.Get)
		candidates.PATCH("/:candidateId", ch.Update)
		candidates.PATCH("/:candidateId/deactivate", ch.Deactivate)
	}

	// ── Exam-scoped enrollment operations ─────────────────────────────────────
	exams := engine.Group("/exams")
	{
		// Phase 1: create enrollment records (no email sent)
		exams.POST("/:examId/enrollments", eh.Enroll)
		exams.GET("/:examId/enrollments", eh.ListByExam)

		// Phase 2: admin explicitly triggers notification emails
		exams.POST("/:examId/enrollments/notify", eh.NotifyAll)

		// Monitoring
		exams.GET("/:examId/sessions", mh.ListSessions)
		exams.GET("/:examId/submissions", mh.GetSubmissions)
	}

	// ── Enrollment-level management ────────────────────────────────────────────
	enrollments := engine.Group("/enrollments")
	{
		enrollments.GET("/:enrollmentId", eh.Get)
		enrollments.PATCH("/:enrollmentId/revoke", eh.Revoke)
		enrollments.POST("/:enrollmentId/reset-attempts", eh.ResetAttempts)

		// Phase 2 (single): admin resends/sends notification to one candidate
		enrollments.POST("/:enrollmentId/notify", eh.Notify)

		// Phase 4: admin fetches a fresh invitation URL for no-email candidates
		enrollments.GET("/:enrollmentId/link", eh.GetLink)
	}

	// ── Access / candidate authentication ────────────────────────────────────
	access := engine.Group("/access")
	{
		// Legacy: validate a raw JWT header directly (internal / gateway use)
		access.POST("/validate", sh.ValidateAccess)

		// Phase 3: candidate exchanges opaque invitation code → receives JWT in body
		access.POST("/redeem", eh.RedeemCode)
	}

	// ── Exam session lifecycle ─────────────────────────────────────────────────
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

	// ── Submission monitoring ─────────────────────────────────────────────────
	submissions := engine.Group("/submissions")
	{
		submissions.GET("/:submissionId", mh.GetSubmissionDetail)
	}

	// ── Internal service-to-service endpoints ─────────────────────────────────
	internal := engine.Group("/internal")
	{
		internal.GET("/candidates/emails", ch.GetEmailsForExam)
	}

	// Swagger UI — available at /swagger/index.html
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return engine
}

// healthCheck returns 200 OK if the service is running.
//
//	@Summary		Health check
//	@ID				healthCheck
//	@Description	Returns a simple JSON indicating the service is alive.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Service is healthy"
//	@Router			/health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "candidate-service"})
}
