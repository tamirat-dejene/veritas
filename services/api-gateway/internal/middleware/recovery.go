package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := "-"
				if value, ok := r.Context().Value(RequestIDKey).(string); ok && value != "" {
					requestID = value
				}
				zap.L().Error("panic recovered", zap.String("request_id", requestID), zap.Any("panic", rec))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
