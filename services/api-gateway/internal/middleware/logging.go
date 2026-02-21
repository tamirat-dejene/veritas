package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tamirat-dejene/veritas/shared/pkg/logger"
	"go.uber.org/zap"
)

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
)

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		requestID := getRequestID(c)
		clientIP := c.ClientIP()
		userID, role, _ := getUserInfo(c)
		status := c.Writer.Status()
		bytes := c.Writer.Size()

		if logger.IsConsoleFormat() {
			logStyle := logger.GetLogStyle()

			switch logStyle {
			case "detailed":
				logDetailed(c, status, duration, requestID, clientIP, userID, role)
			case "minimal":
				logMinimal(c, status, duration)
			default: // compact
				logCompact(c, status, duration, clientIP, userID, role)
			}
			return
		}

		// JSON format - structured logging
		zap.L().Info(
			"request_completed",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Int("bytes", bytes),
			zap.String("ip", clientIP),
			zap.String("user_id", userID),
			zap.String("role", role),
		)
	}
}

func logCompact(c *gin.Context, status int, duration time.Duration, clientIP, userID, role string) {
	statusColor := getStatusColor(status)
	methodColor := getMethodColor(c.Request.Method)

	// Format: INFO GET /path → 200 OK (5ms) [IP] user:john role:admin
	message := fmt.Sprintf(
		"%s%s%s %s → %s%d %s%s %s(%s)%s [%s]",
		methodColor, c.Request.Method, colorReset,
		c.Request.URL.Path,
		statusColor, status, httpStatusText(status), colorReset,
		colorGray, formatDuration(duration), colorReset,
		clientIP,
	)

	// Add user info if authenticated
	if userID != "-" {
		message += fmt.Sprintf(" %suser:%s%s", colorCyan, userID, colorReset)
	}
	if role != "-" {
		message += fmt.Sprintf(" %srole:%s%s", colorBlue, role, colorReset)
	}

	zap.L().Info(message)
}

func logDetailed(c *gin.Context, status int, duration time.Duration, requestID, clientIP, userID, role string) {
	statusColor := getStatusColor(status)
	methodColor := getMethodColor(c.Request.Method)

	message := fmt.Sprintf(
		"\n%s┌─ REQUEST ────────────────────────────────────────%s\n"+
			"│ %s%s %s%s\n"+
			"│ Status: %s%d %s%s\n"+
			"│ Duration: %s\n"+
			"│ IP: %s\n"+
			"│ Request ID: %s\n",
		colorGray, colorReset,
		methodColor, c.Request.Method, c.Request.URL.Path, colorReset,
		statusColor, status, httpStatusText(status), colorReset,
		formatDuration(duration),
		clientIP,
		requestID,
	)

	if userID != "-" {
		message += fmt.Sprintf("│ User: %s (Role: %s)\n", userID, role)
	}

	message += fmt.Sprintf("%s└──────────────────────────────────────────────────%s", colorGray, colorReset)

	zap.L().Info(message)
}

func logMinimal(c *gin.Context, status int, duration time.Duration) {
	statusColor := getStatusColor(status)

	// Format: GET /path 200 (5ms)
	message := fmt.Sprintf(
		"%s %s %s%d%s (%s)",
		c.Request.Method,
		c.Request.URL.Path,
		statusColor, status, colorReset,
		formatDuration(duration),
	)

	zap.L().Info(message)
}

func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return colorGreen
	case status >= 300 && status < 400:
		return colorYellow
	case status >= 400:
		return colorRed
	default:
		return colorReset
	}
}

func getMethodColor(method string) string {
	switch method {
	case "GET":
		return colorCyan
	case "POST":
		return colorGreen
	case "PUT", "PATCH":
		return colorYellow
	case "DELETE":
		return colorRed
	default:
		return colorReset
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func getRequestID(c *gin.Context) string {
	if requestID := c.GetString(RequestIDKey); requestID != "" {
		return requestID
	}
	return "-"
}

func getTenantID(c *gin.Context) string {
	if tenantID := c.GetString(TenantIDKey); tenantID != "" {
		return tenantID
	}
	return "-"
}

func getUserInfo(c *gin.Context) (string, string, string) {
	claimsVal, exists := c.Get(string(UserContextKey))
	if !exists {
		return "-", "-", "-"
	}

	claims, ok := claimsVal.(*UserClaims)
	if !ok || claims == nil {
		return "-", "-", "-"
	}

	userID := claims.UserID
	role := string(claims.Role)
	enterpriseID := claims.EnterpriseID

	if userID == "" {
		userID = "-"
	}
	if role == "" {
		role = "-"
	}
	if enterpriseID == "" {
		enterpriseID = "-"
	}

	return userID, role, enterpriseID
}

func httpStatusText(status int) string {
	// Simple mapping for common statuses to avoid importing net/http just for this
	// Alternatively can just import net/http
	switch status {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}
