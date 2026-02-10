package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tamirat-dejene/veritas/services/api-gateway/internal/logger"
	"go.uber.org/zap"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		requestID := getRequestID(r)
		clientIP := getClientIP(r)
		userAgent := r.UserAgent()
		userID, role, enterpriseID := getUserInfo(r)
		tenantID := getTenantID(r)
		query := r.URL.RawQuery

		if logger.IsConsoleFormat() {
			message := fmt.Sprintf(
				"request_id=%s method=%s path=%s status=%d duration_ms=%d bytes=%d ip=%s user_id=%s role=%s enterprise_id=%s tenant_id=%s ua=%q query=%q",
				requestID,
				r.Method,
				r.URL.Path,
				rw.status,
				time.Since(start).Milliseconds(),
				rw.bytes,
				clientIP,
				userID,
				role,
				enterpriseID,
				tenantID,
				userAgent,
				query,
			)
			zap.L().Info(message)
			return
		}

		zap.L().Info(
			"request_completed",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.status),
			zap.Int64("duration_ms", time.Since(start).Milliseconds()),
			zap.Int("bytes", rw.bytes),
			zap.String("ip", clientIP),
			zap.String("user_agent", userAgent),
			zap.String("query", query),
			zap.String("user_id", userID),
			zap.String("role", role),
			zap.String("enterprise_id", enterpriseID),
			zap.String("tenant_id", tenantID),
		)
	})
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
