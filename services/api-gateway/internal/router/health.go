package router

import (
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/config"
)

// RegisterHealthCheck sets up the detailed health check endpoint
func (g *RouterGroup) RegisterHealthCheck(cfg *config.Config) {
	g.engine.GET("/health", func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept"), "text/html") {
			c.String(http.StatusOK, "OK")
			return
		}

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
				Title: "Backbone Infrastructure",
				Services: []struct {
					Name string
					URL  string
					Type string
				}{
					{"PostgreSQL Database", cfg.DatabaseURL, "postgres"},
					{"Redis Cache", cfg.RedisAddr, "redis"},
				},
			},
		}

		type CategoryResult struct {
			Title    string
			Services []ServiceHealth
		}

		var wg sync.WaitGroup
		results := make([]CategoryResult, len(categories))
		client := &http.Client{Timeout: 2 * time.Second}

		for catIdx, cat := range categories {
			results[catIdx].Title = cat.Title
			results[catIdx].Services = make([]ServiceHealth, len(cat.Services))

			for svcIdx, svc := range cat.Services {
				wg.Add(1)
				go func(catIdx, svcIdx int, name, url, svcType string) {
					defer wg.Done()
					start := time.Now()
					status := "DOWN"
					errMsg := ""

					switch svcType {
					case "http":
						resp, err := client.Get(url + "/health")
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
					case "postgres":
						hostPort := strings.TrimPrefix(url, "postgres://")
						hostPort = strings.Split(hostPort, "/")[0]
						hostPort = strings.Split(hostPort, "@")[len(strings.Split(hostPort, "@"))-1]
						if !strings.Contains(hostPort, ":") {
							hostPort += ":5432"
						}
						conn, err := net.DialTimeout("tcp", hostPort, 2*time.Second)
						if err == nil {
							status = "UP"
							conn.Close()
						} else {
							errMsg = err.Error()
						}
					case "redis":
						conn, err := net.DialTimeout("tcp", url, 2*time.Second)
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
						URL:       url,
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
			// Cannot change status here easily if already sent via template partial, but template err log is ok
			c.Error(err)
		}
	})
}
