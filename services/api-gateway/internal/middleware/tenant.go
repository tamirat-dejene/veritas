package middleware

import (
	"context"
	"net/http"
)

type contextKeyTenant string

const TenantIDKey contextKeyTenant = "tenantID"

// TenantResolver middleware extracts the enterpriseId from the JWT claims
// and injects it into the request context.
// This relies on JWTAuth middleware being run beforehand.
func TenantResolver(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get claims from context (populated by JWTAuth)
		claims, ok := r.Context().Value(UserContextKey).(*UserClaims)
		if !ok {
			// If no claims (e.g., public route), proceed without tenant ID
			// OR we could decide to error out if tenant is strictly required.
			// For now, let's proceed. The route handler or RBAC will decide if auth is needed.
			next.ServeHTTP(w, r)
			return
		}

		if claims.EnterpriseID != "" {
			ctx := context.WithValue(r.Context(), TenantIDKey, claims.EnterpriseID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			// If logged in but no enterprise ID, just proceed.
			next.ServeHTTP(w, r)
		}
	})
}
