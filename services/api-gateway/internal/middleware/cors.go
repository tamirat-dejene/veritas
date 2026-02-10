package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins, allowedMethods, allowedHeaders []string) func(http.Handler) http.Handler {
	allowAllOrigins := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowedOriginSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "" || origin == "*" {
			continue
		}
		allowedOriginSet[origin] = struct{}{}
	}

	allowMethods := strings.Join(allowedMethods, ", ")
	allowHeaders := strings.Join(allowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAllOrigins {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else if _, ok := allowedOriginSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if allowMethods != "" {
					w.Header().Set("Access-Control-Allow-Methods", allowMethods)
				}
				if allowHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
