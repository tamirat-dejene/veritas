package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/infrastructure"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/middleware"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/proxy"
	smw "github.com/tamirat-dejene/veritas/shared/pkg/middleware"
)

func NewRouter(cfg *config.Config, rateLimiter domain.RateLimiter) (http.Handler, error) {
	// Initialize Gin engine without default logger/recovery so we can use our custom ones
	if cfg.SystemMode == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()

	// --- Global Middleware ---

	parseCSV := func(value string) []string {
		parts := strings.Split(value, ",")
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			item := strings.TrimSpace(part)
			if item != "" {
				items = append(items, item)
			}
		}
		return items
	}

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)
	corsMiddleware := smw.CORS(
		parseCSV(cfg.CORSAllowedOrigins),
		parseCSV(cfg.CORSAllowedMethods),
		parseCSV(cfg.CORSAllowedHeaders),
	)

	engine.Use(smw.RequestID())
	engine.Use(smw.Logging())
	engine.Use(corsMiddleware)
	engine.Use(smw.Recovery())
	engine.Use(rateLimitMiddleware.Handler())

	// --- Set up Router Group ---
	routerGroup := NewRouterGroup(engine, cfg.JWTSecret)

	// --- Circuit Breakers ---
	cbSettings := infrastructure.DefaultCircuitBreakerSettings()

	authCB := infrastructure.NewCircuitBreaker("auth-service", cbSettings)
	enterpriseCB := infrastructure.NewCircuitBreaker("enterprise-service", cbSettings)
	paymentCB := infrastructure.NewCircuitBreaker("payment-service", cbSettings)
	examCB := infrastructure.NewCircuitBreaker("exam-service", cbSettings)
	candidateCB := infrastructure.NewCircuitBreaker("candidate-service", cbSettings)
	proctoringCB := infrastructure.NewCircuitBreaker("proctoring-service", cbSettings)
	faceCB := infrastructure.NewCircuitBreaker("face-verification-service", cbSettings)
	gradingCB := infrastructure.NewCircuitBreaker("grading-service", cbSettings)
	reportingCB := infrastructure.NewCircuitBreaker("reporting-service", cbSettings)

	// --- Service Proxies ---
	authProxy, err := proxy.NewProxy(cfg.AuthServiceURL, authCB, "auth-service")
	if err != nil {
		return nil, err
	}
	enterpriseProxy, err := proxy.NewProxy(cfg.EnterpriseServiceURL, enterpriseCB, "enterprise-service")
	if err != nil {
		return nil, err
	}
	paymentProxy, err := proxy.NewProxy(cfg.PaymentServiceURL, paymentCB, "payment-service")
	if err != nil {
		return nil, err
	}
	examProxy, err := proxy.NewProxy(cfg.ExamServiceURL, examCB, "exam-service")
	if err != nil {
		return nil, err
	}
	candidateProxy, err := proxy.NewProxy(cfg.CandidateServiceURL, candidateCB, "candidate-service")
	if err != nil {
		return nil, err
	}
	proctoringProxy, err := proxy.NewProxy(cfg.ProctoringServiceURL, proctoringCB, "proctoring-service")
	if err != nil {
		return nil, err
	}
	faceProxy, err := proxy.NewProxy(cfg.FaceVerificationServiceURL, faceCB, "face-verification-service")
	if err != nil {
		return nil, err
	}
	gradingProxy, err := proxy.NewProxy(cfg.GradingServiceURL, gradingCB, "grading-service")
	if err != nil {
		return nil, err
	}
	reportingProxy, err := proxy.NewProxy(cfg.ReportingServiceURL, reportingCB, "reporting-service")
	if err != nil {
		return nil, err
	}

	// --- Route Attachments ---

	routerGroup.RegisterHealthCheck(cfg)
	if err := routerGroup.RegisterDocs(); err != nil {
		return nil, err
	}
	routerGroup.RegisterSwaggerRoutes(map[string]http.Handler{
		"auth":       authProxy,
		"candidate":  candidateProxy,
		"enterprise": enterpriseProxy,
		"exam":       examProxy,
		"payment":    paymentProxy,
	})

	routerGroup.RegisterAuthRoutes(authProxy)
	routerGroup.RegisterEnterpriseRoutes(enterpriseProxy)
	routerGroup.RegisterPaymentRoutes(paymentProxy)
	routerGroup.RegisterExamRoutes(examProxy)
	routerGroup.RegisterCandidateRoutes(candidateProxy)
	routerGroup.RegisterProctoringRoutes(proctoringProxy)
	routerGroup.RegisterFaceVerificationRoutes(faceProxy)
	routerGroup.RegisterGradingRoutes(gradingProxy)
	routerGroup.RegisterReportingRoutes(reportingProxy)

	return engine, nil
}
