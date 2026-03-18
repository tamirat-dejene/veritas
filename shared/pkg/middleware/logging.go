package middleware

import (
	"fmt"
	"net/http"
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
)

// Logging returns a middleware that logs HTTP requests.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		requestID := getContextString(c, string(logger.RequestIDKey))
		userID := getContextString(c, string(logger.UserIDKey))
		role := getContextString(c, string(logger.RoleKey))
		
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		if logger.IsConsoleFormat() {
			logStyle := logger.GetLogStyle()
			switch logStyle {
			case "minimal":
				logMinimal(method, path, status, duration)
			default: // compact or detailed (compact is default)
				logCompact(method, path, status, duration, clientIP, userID, role)
			}
			return
		}

		// JSON format - structured logging
		zap.L().Info(
			"request_completed",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("ip", clientIP),
			zap.String("user_id", userID),
			zap.String("role", role),
		)
	}
}

func logCompact(method, path string, status int, duration time.Duration, clientIP, userID, role string) {
	statusColor := getStatusColor(status)
	methodColor := getMethodColor(method)

	message := fmt.Sprintf(
		"%s%s%s %s → %s%d %s%s %s(%s)%s [%s]",
		methodColor, method, colorReset,
		path,
		statusColor, status, http.StatusText(status), colorReset,
		colorGray, formatDuration(duration), colorReset,
		clientIP,
	)

	if userID != "" && userID != "-" {
		message += fmt.Sprintf(" %suser:%s%s", colorCyan, userID, colorReset)
	}
	if role != "" && role != "-" {
		message += fmt.Sprintf(" %srole:%s%s", colorBlue, role, colorReset)
	}

	zap.L().Info(message)
}

func logMinimal(method, path string, status int, duration time.Duration) {
	statusColor := getStatusColor(status)
	message := fmt.Sprintf(
		"%s %s %s%d%s (%s)",
		method, path, statusColor, status, colorReset, formatDuration(duration),
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

func getContextString(c *gin.Context, key string) string {
	if val, exists := c.Get(key); exists {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return "-"
}
