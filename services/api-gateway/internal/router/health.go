package router

import (
	"context"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/domain"
)

// RegisterHealthCheck sets up health check endpoints.
func (g *RouterGroup) RegisterHealthCheck(cfg *config.Config) {
	// Public liveness probe — no auth required, minimal response.
	g.engine.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Detailed diagnostic view — restricted to SystemAdmin.
	detailChain := append(
		g.authWithRoles(domain.RoleSystemAdmin),
		func(c *gin.Context) { g.handleHealthDetail(c, cfg) },
	)
	g.engine.GET("/health/detail", detailChain...)
}

func (g *RouterGroup) handleHealthDetail(c *gin.Context, cfg *config.Config) {
	type ServiceHealth struct {
		Name      string
		URL       string
		Status    string
		Latency   string
		Error     string
		Timestamp string
	}

	type Category struct {
		Title    string
		Services []struct {
			Name string
			URL  string
			Type string // "http", "redis", "postgres"
		}
	}

	categories := []Category{
		{
			Title: "Go Distributed Services",
			Services: []struct {
				Name string
				URL  string
				Type string
			}{
				{"Auth Service", cfg.AuthServiceURL, "http"},
				{"Enterprise Service", cfg.EnterpriseServiceURL, "http"},
				{"Payment Service", cfg.PaymentServiceURL, "http"},
				{"Exam Service", cfg.ExamServiceURL, "http"},
				{"Candidate Service", cfg.CandidateServiceURL, "http"},
			},
		},
		{
			Title: "Python AI & Logic Services",
			Services: []struct {
				Name string
				URL  string
				Type string
			}{
				{"Proctoring Service", cfg.ProctoringServiceURL, "http"},
				{"Face Verification Service", cfg.FaceVerificationServiceURL, "http"},
				{"Grading Service", cfg.GradingServiceURL, "http"},
				{"Reporting Service", cfg.ReportingServiceURL, "http"},
			},
		},
		{
			Title: "Infrastructure",
			Services: []struct {
				Name string
				URL  string
				Type string
			}{
				// BUG-09 FIX: Redact DB credentials before exposing in the health page.
				{"PostgreSQL Database", redactDatabaseURL(cfg.DatabaseURL), "postgres"},
				{"Redis Cache", cfg.RedisHost + ":" + strconv.Itoa(cfg.RedisPort), "redis"},
				{"Kafka Broker", func() string {
					if len(cfg.KafkaBrokers) > 0 {
						return cfg.KafkaBrokers[0]
					}
					return "kafka:9092"
				}(), "kafka"},
			},
		},
	}

	type CategoryResult struct {
		Title    string
		Services []ServiceHealth
	}

	probeCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]CategoryResult, len(categories))
	client := &http.Client{Timeout: 2 * time.Second}

	for catIdx, cat := range categories {
		results[catIdx].Title = cat.Title
		results[catIdx].Services = make([]ServiceHealth, len(cat.Services))

		for svcIdx, svc := range cat.Services {
			wg.Add(1)
			go func(catIdx, svcIdx int, name, rawURL, svcType string) {
				defer wg.Done()
				start := time.Now()
				status := "DOWN"
				errMsg := ""

				switch svcType {
				case "http":
					req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, rawURL+"/health", nil)
					if err == nil {
						resp, err := client.Do(req)
						if err == nil {
							if resp.StatusCode == http.StatusOK {
								status = "UP"
							} else {
								errMsg = "HTTP " + resp.Status
							}
							resp.Body.Close()
						} else {
							errMsg = err.Error()
						}
					}
				case "postgres":
					// rawURL is already redacted, parse host:port for TCP probe
					dialURL := strings.TrimPrefix(rawURL, "postgres://")
					dialURL = strings.Split(dialURL, "/")[0]
					dialURL = strings.Split(dialURL, "@")[len(strings.Split(dialURL, "@"))-1]
					if !strings.Contains(dialURL, ":") {
						dialURL += ":5432"
					}
					var d net.Dialer
					conn, err := d.DialContext(probeCtx, "tcp", dialURL)
					if err == nil {
						status = "UP"
						conn.Close()
					} else {
						errMsg = err.Error()
					}
				case "redis", "kafka":
					var d net.Dialer
					conn, err := d.DialContext(probeCtx, "tcp", rawURL)
					if err == nil {
						status = "UP"
						conn.Close()
					} else {
						errMsg = err.Error()
					}
				}

				latency := time.Since(start).String()
				results[catIdx].Services[svcIdx] = ServiceHealth{
					Name:      name,
					URL:       rawURL,
					Status:    status,
					Latency:   latency,
					Error:     errMsg,
					Timestamp: time.Now().Format(time.RFC1123),
				}
			}(catIdx, svcIdx, svc.Name, svc.URL, svc.Type)
		}
	}

	wg.Wait()

	tmpl, err := template.New("health.html").Funcs(template.FuncMap{
		"stringsContains": strings.Contains,
	}).ParseFS(docsFS, "docs/pages/health.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load health template: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/html")
	err = tmpl.Execute(c.Writer, map[string]interface{}{
		"Categories": results,
	})
	if err != nil {
		c.Error(err)
	}
}
