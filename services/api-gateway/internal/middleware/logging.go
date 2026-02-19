package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

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

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		requestID := getRequestID(r)
		clientIP := getClientIP(r)
		userID, role, _ := getUserInfo(r)

		if logger.IsConsoleFormat() {
			logStyle := logger.GetLogStyle()

			switch logStyle {
			case "detailed":
				logDetailed(r, rw, duration, requestID, clientIP, userID, role)
			case "minimal":
				logMinimal(r, rw, duration)
			default: // compact
				logCompact(r, rw, duration, clientIP, userID, role)
			}
			return
		}

		// JSON format - structured logging
		zap.L().Info(
			"request_completed",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.status),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Int("bytes", rw.bytes),
			zap.String("ip", clientIP),
			zap.String("user_id", userID),
			zap.String("role", role),
		)
	})
}

func logCompact(r *http.Request, rw *responseWriter, duration time.Duration, clientIP, userID, role string) {
	statusColor := getStatusColor(rw.status)
	methodColor := getMethodColor(r.Method)

	// Format: INFO GET /path → 200 OK (5ms) [IP] user:john role:admin
	message := fmt.Sprintf(
		"%s%s%s %s → %s%d %s%s %s(%s)%s [%s]",
		methodColor, r.Method, colorReset,
		r.URL.Path,
		statusColor, rw.status, http.StatusText(rw.status), colorReset,
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

func logDetailed(r *http.Request, rw *responseWriter, duration time.Duration, requestID, clientIP, userID, role string) {
	statusColor := getStatusColor(rw.status)
	methodColor := getMethodColor(r.Method)

	message := fmt.Sprintf(
		"\n%s┌─ REQUEST ────────────────────────────────────────%s\n"+
			"│ %s%s %s%s\n"+
			"│ Status: %s%d %s%s\n"+
			"│ Duration: %s\n"+
			"│ IP: %s\n"+
			"│ Request ID: %s\n",
		colorGray, colorReset,
		methodColor, r.Method, r.URL.Path, colorReset,
		statusColor, rw.status, http.StatusText(rw.status), colorReset,
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

func logMinimal(r *http.Request, rw *responseWriter, duration time.Duration) {
	statusColor := getStatusColor(rw.status)

	// Format: GET /path 200 (5ms)
	message := fmt.Sprintf(
		"%s %s %s%d%s (%s)",
		r.Method,
		r.URL.Path,
		statusColor, rw.status, colorReset,
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

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(p)
	rw.bytes += n
	return n, err
}

func getRequestID(r *http.Request) string {
	if requestID, ok := r.Context().Value(RequestIDKey).(string); ok && requestID != "" {
		return requestID
	}
	return "-"
}

func getTenantID(r *http.Request) string {
	if tenantID, ok := r.Context().Value(TenantIDKey).(string); ok && tenantID != "" {
		return tenantID
	}
	return "-"
}

func getUserInfo(r *http.Request) (string, string, string) {
	claims, ok := r.Context().Value(UserContextKey).(*UserClaims)
	if !ok || claims == nil {
		return "-", "-", "-"
	}

	userID := claims.UserID
	role := claims.Role
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

func getClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}

	return "-"
}
