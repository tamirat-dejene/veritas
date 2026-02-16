package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/tamirat-dejene/veritas/services/auth-service/internal/application"
)

type AuthHandler struct {
	authService *application.AuthService
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Role, req.FirstName, req.LastName)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrUserExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, application.ErrInvalidEmail),
			errors.Is(err, application.ErrInvalidPassword),
			errors.Is(err, application.ErrInvalidRole),
			errors.Is(err, application.ErrMissingName):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidEmail), errors.Is(err, application.ErrInvalidPassword):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, application.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{Token: token})
}

func (h *AuthHandler) Validate(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, "authorization header required")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		writeError(w, http.StatusUnauthorized, "invalid token format")
		return
	}

	user, err := h.authService.Validate(r.Context(), tokenString)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidToken):
			writeError(w, http.StatusUnauthorized, err.Error())
		case errors.Is(err, application.ErrUserNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
