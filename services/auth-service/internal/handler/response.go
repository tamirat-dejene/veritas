package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the JSON body for all error responses.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid email or password"`
}

// TokenResponse is the JSON body returned upon successful login or refresh.
type TokenResponse struct {
	AccessToken  string `json:"accessToken"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refreshToken" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpiresIn    int64  `json:"expiresIn"    example:"3600"`
}

// writeError writes a JSON error body with the given HTTP status.
// Generic messages are preferred for security-sensitive responses (no enumeration).
func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Error: message})
}

// writeTokens writes a successful token pair response.
func writeTokens(c *gin.Context, accessToken, refreshToken string, expiresIn int64) {
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}
