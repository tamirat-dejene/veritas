package http

import (
	"net/http"

	"github.com/tamirat-dejene/veritas/services/auth-service/internal/application"
)

func NewRouter(authService *application.AuthService) http.Handler {
	handler := NewAuthHandler(authService)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", handler.Register)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.HandleFunc("POST /auth/validate", handler.Validate) // Optional, for internal verification

	return mux
}
