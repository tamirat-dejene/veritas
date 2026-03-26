package router

import (
	"net/http"

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
	sysAdmin := g.authWithRoles(domain.RoleSystemAdmin)
	entAdmin := g.authWithRoles(domain.RoleEnterpriseAdmin)
	sysOrEntAdmin := g.authWithRoles(domain.RoleSystemAdmin, domain.RoleEnterpriseAdmin)
	allAuth := g.authWithRoles(domain.RoleAll)

	// Registration (anyone can register an enterprise)
	g.register("POST", "/enterprises", proxy)

	// Discovery
	g.register("GET", "/enterprises", proxy, sysAdmin...)
	g.register("GET", "/enterprises/me", proxy, entAdmin...)
	g.register("GET", "/enterprises/slug/:slug", proxy, allAuth...) // internal gateway routing

	// Single enterprise read & general update
	g.register("GET", "/enterprises/:enterpriseId", proxy, sysOrEntAdmin...)
	g.register("PATCH", "/enterprises/:enterpriseId", proxy, entAdmin...)

	// Admin lifecycle
	g.register("POST", "/enterprises/:enterpriseId/approve", proxy, sysAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/suspend", proxy, sysAdmin...)
	g.register("DELETE", "/enterprises/:enterpriseId", proxy, sysAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/reactivate", proxy, sysAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/restore", proxy, sysAdmin...)
	g.register("DELETE", "/enterprises/:enterpriseId/permanent", proxy, sysAdmin...)

	// Self-service branding & settings (owner-scoped)
	g.register("PATCH", "/enterprises/:enterpriseId/branding", proxy, entAdmin...)
	g.register("PATCH", "/enterprises/:enterpriseId/settings", proxy, entAdmin...)

	// Status, domain validation, audit
	g.register("GET", "/enterprises/:enterpriseId/status", proxy, sysOrEntAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/validate-domain", proxy, entAdmin...)
	g.register("GET", "/enterprises/:enterpriseId/summary", proxy, sysOrEntAdmin...)
	g.register("GET", "/enterprises/:enterpriseId/audit-logs", proxy, sysOrEntAdmin...)

	// Subscription management
	g.register("POST", "/enterprises/:enterpriseId/subscription", proxy, sysAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/subscription/cancel", proxy, sysAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/subscription/renew", proxy, sysAdmin...)
	g.register("GET", "/enterprises/:enterpriseId/subscription", proxy, sysOrEntAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/suspend-payment", proxy, sysAdmin...)

	// Enterprise user management
	g.register("POST", "/enterprises/:enterpriseId/users", proxy, entAdmin...)
	g.register("GET", "/enterprises/:enterpriseId/users", proxy, entAdmin...)
	g.register("GET", "/enterprises/:enterpriseId/users/:userId", proxy, entAdmin...)
	g.register("PATCH", "/enterprises/:enterpriseId/users/:userId", proxy, entAdmin...)
	g.register("PATCH", "/enterprises/:enterpriseId/users/:userId/deactivate", proxy, entAdmin...)
	g.register("POST", "/enterprises/:enterpriseId/users/:userId/reset-password", proxy, entAdmin...)
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
	g.register("GET", "/exams/:examId/questions", proxy, adminRole...)
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
	adminRole := g.authWithRoles(domain.RoleEnterpriseAdmin)
	staffOrAdmin := g.authWithRoles(domain.RoleEnterpriseAdmin, domain.RoleEnterpriseStaff)
	candidateRole := g.candidateAuthChain()
	adminOrAuto := g.authWithRoles(domain.RoleEnterpriseAdmin, domain.RoleEnterpriseAuto)

	// Candidates
	g.register("POST", "/candidates", proxy, adminOrAuto...)
	g.register("POST", "/candidates/bulk", proxy, adminRole...)
	g.register("GET", "/candidates", proxy, staffOrAdmin...)
	g.register("GET", "/candidates/:candidateId", proxy, staffOrAdmin...)
	g.register("PATCH", "/candidates/:candidateId", proxy, adminRole...)
	g.register("PATCH", "/candidates/:candidateId/deactivate", proxy, adminRole...)

	// Enrollments
	g.register("POST", "/exams/:examId/enrollments", proxy, staffOrAdmin...)
	g.register("GET", "/exams/:examId/enrollments", proxy, staffOrAdmin...)
	g.register("GET", "/exams/:examId/sessions", proxy, adminRole...)
	g.register("GET", "/exams/:examId/submissions", proxy, adminRole...)
	g.register("GET", "/enrollments/:enrollmentId", proxy, staffOrAdmin...)
	g.register("POST", "/enrollments/:enrollmentId/regenerate-token", proxy, adminRole...)
	g.register("PATCH", "/enrollments/:enrollmentId/revoke", proxy, adminRole...)
	g.register("POST", "/enrollments/:enrollmentId/reset-attempts", proxy, adminRole...)

	// Access & Sessions
	g.register("POST", "/access/validate", proxy) // Public
	g.register("POST", "/sessions/start", proxy, candidateRole...)
	g.register("GET", "/sessions/me/active", proxy, candidateRole...)
	g.register("GET", "/sessions/:sessionId", proxy, g.candidateOrAdminChain(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/sessions/:sessionId/questions", proxy, candidateRole...)
	g.register("PATCH", "/sessions/:sessionId/answers", proxy, candidateRole...)
	g.register("GET", "/sessions/:sessionId/answers", proxy, candidateRole...)
	g.register("POST", "/sessions/:sessionId/submit", proxy, candidateRole...)
	g.register("POST", "/sessions/:sessionId/terminate", proxy, adminRole...)
	g.register("POST", "/sessions/:sessionId/expire", proxy, adminOrAuto...)
	g.register("GET", "/sessions/:sessionId/summary", proxy, adminRole...)
	g.register("GET", "/sessions/:sessionId/result", proxy, candidateRole...)

	// Submissions
	g.register("GET", "/submissions/:submissionId", proxy, adminRole...)
}


// RegisterProctoringRoutes attaches Proctoring Service proxy routes
func (g *RouterGroup) RegisterProctoringRoutes(proxy http.Handler) {
	candidateRole := g.candidateAuthChain()
	g.register("POST", "/proctoring/events", proxy, candidateRole...)
	g.register("GET", "/proctoring/sessions/:sessionId/events", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
}

// RegisterFaceVerificationRoutes attaches Face Verification Service proxy routes
func (g *RouterGroup) RegisterFaceVerificationRoutes(proxy http.Handler) {
	premiumCandidateBlock := append(g.candidateAuthChain(), middleware.RequireTier(domain.TierEnterprise))
	g.register("POST", "/face/register", proxy, premiumCandidateBlock...)
	g.register("POST", "/face/verify", proxy, premiumCandidateBlock...)
}

// RegisterGradingRoutes attaches Grading Service proxy routes
func (g *RouterGroup) RegisterGradingRoutes(proxy http.Handler) {
	g.register("POST", "/grading/auto", proxy, g.authWithRoles(domain.RoleEnterpriseAuto)...)
	g.register("POST", "/grading/manual", proxy, g.authWithRoles(domain.RoleEnterpriseStaff)...)
	g.register("GET", "/results/:examId", proxy, g.authWithRoles(domain.RoleEnterpriseAdmin)...)
	g.register("GET", "/certificates/:certificateId", proxy, g.candidateAuthChain()...)
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
