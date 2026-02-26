package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/middleware"
)

// RegisterAuthRoutes attaches Auth Service proxy routes
func (g *RouterGroup) RegisterAuthRoutes(proxy http.Handler) {
	g.register("POST", "/auth/login", proxy)
	g.register("POST", "/auth/refresh", proxy)
	g.register("POST", "/auth/logout", proxy, g.authWithRoles(domain.RoleAll)...)
}

// RegisterEnterpriseRoutes attaches Enterprise Service proxy routes
func (g *RouterGroup) RegisterEnterpriseRoutes(proxy http.Handler) {
	g.register("POST", "/enterprises", proxy)
	g.register("POST", "/enterprises/:enterpriseId/approve", proxy, g.authWithRoles(domain.RoleSystemAdmin)...)
	g.register("PATCH", "/enterprises/:enterpriseId", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("POST", "/enterprises/:enterpriseId/suspend", proxy, g.authWithRoles(domain.RoleSystemAdmin)...)
	g.register("DELETE", "/enterprises/:enterpriseId", proxy, g.authWithRoles(domain.RoleSystemAdmin)...)
}

// RegisterPaymentRoutes attaches Payment Service proxy routes
func (g *RouterGroup) RegisterPaymentRoutes(proxy http.Handler) {
	g.register("GET", "/subscriptions/plans", proxy)
	g.register("POST", "/subscriptions/:enterpriseId/upgrade", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/payments/history", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/invoices/:invoiceId", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("POST", "/webhooks/stripe", proxy) // Public webhook
}

// RegisterExamRoutes attaches Exam Service proxy routes
func (g *RouterGroup) RegisterExamRoutes(proxy http.Handler) {
	staffOrAdmin := g.authWithRoles(domain.RoleEnterpriseAdmin, domain.RoleEnterpriseStaff)
	adminRole := g.authWithRoles(domain.RoleEnterpriseAdmin)

	// Questions
	g.register("POST", "/questions", proxy, staffOrAdmin...)
	g.register("GET", "/questions", proxy, staffOrAdmin...)
	g.register("GET", "/questions/:questionId", proxy, staffOrAdmin...)
	g.register("PATCH", "/questions/:questionId", proxy, staffOrAdmin...)
	g.register("DELETE", "/questions/:questionId", proxy, staffOrAdmin...)

	// Exams Lifecycle
	g.register("POST", "/exams", proxy, adminRole...)
	g.register("GET", "/exams", proxy, adminRole...)
	g.register("GET", "/exams/:examId", proxy, adminRole...)
	g.register("PATCH", "/exams/:examId", proxy, adminRole...)
	g.register("POST", "/exams/:examId/schedule", proxy, adminRole...)
	g.register("POST", "/exams/:examId/clone", proxy, adminRole...)
	g.register("POST", "/exams/:examId/publish", proxy, adminRole...)
	g.register("POST", "/exams/:examId/close", proxy, adminRole...)
	g.register("DELETE", "/exams/:examId", proxy, adminRole...)

	// Exam Questions Assembly
	g.register("POST", "/exams/:examId/questions", proxy, adminRole...)
	g.register("DELETE", "/exams/:examId/questions/:questionId", proxy, adminRole...)
	g.register("PATCH", "/exams/:examId/questions/:questionId", proxy, adminRole...)

	// Exam Randomization Rules
	g.register("POST", "/exams/:examId/rules", proxy, adminRole...)
	g.register("PATCH", "/exams/:examId/rules/:ruleId", proxy, adminRole...)
	g.register("DELETE", "/exams/:examId/rules/:ruleId", proxy, adminRole...)
}

// RegisterCandidateRoutes attaches Candidate Service proxy routes
func (g *RouterGroup) RegisterCandidateRoutes(proxy http.Handler) {
	g.register("POST", "/candidates/bulk", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)

	candidateRole := g.authWithRoles(domain.RoleExamCandidate)
	g.register("POST", "/sessions/start", proxy, candidateRole...)
	g.register("PATCH", "/sessions/:sessionId/answers", proxy, candidateRole...)
	g.register("POST", "/sessions/:sessionId/submit", proxy, candidateRole...)
	g.register("POST", "/sessions/:sessionId/terminate", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
}

// RegisterProctoringRoutes attaches Proctoring Service proxy routes
func (g *RouterGroup) RegisterProctoringRoutes(proxy http.Handler) {
	candidateRole := g.authWithRoles(domain.RoleExamCandidate)
	g.register("POST", "/proctoring/events", proxy, candidateRole...)
	g.register("GET", "/proctoring/sessions/:sessionId/events", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
}

// RegisterFaceVerificationRoutes attaches Face Verification Service proxy routes
func (g *RouterGroup) RegisterFaceVerificationRoutes(proxy http.Handler) {
	requirePremium := func(c *gin.Context) {
		claimsVal, exists := c.Get(string(middleware.UserContextKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Premium subscription required"})
			return
		}

		claims, ok := claimsVal.(*middleware.UserClaims)
		if !ok || claims.Tier != "Premium" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Premium subscription required"})
			return
		}

		c.Next()
	}

	premiumCandidateBlock := append(g.authWithRoles(domain.RoleExamCandidate), requirePremium)
	g.register("POST", "/face/register", proxy, premiumCandidateBlock...)
	g.register("POST", "/face/verify", proxy, premiumCandidateBlock...)
}

// RegisterGradingRoutes attaches Grading Service proxy routes
func (g *RouterGroup) RegisterGradingRoutes(proxy http.Handler) {
	g.register("POST", "/grading/auto", proxy, g.authWithRoles(domain.RoleEnterpriseAuto)...)
	g.register("POST", "/grading/manual", proxy, g.authWithRoles(domain.RoleEnterpriseStaff)...)
	g.register("GET", "/results/:examId", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/certificates/:certificateId", proxy, g.authWithRoles(domain.RoleExamCandidate)...)
}

// RegisterReportingRoutes attaches Reporting Service proxy routes
func (g *RouterGroup) RegisterReportingRoutes(proxy http.Handler) {
	auditRole := g.authWithRoles(domain.RoleSystemAdmin, domain.RoleEnterpriseAdmin)

	g.register("GET", "/dashboard/metrics", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin, domain.RoleEnterpriseStaff)...)
	g.register("GET", "/monitoring/exams/:examId", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("POST", "/reports", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/reports/:reportId/export", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/audit/logs", proxy, auditRole...)
}
