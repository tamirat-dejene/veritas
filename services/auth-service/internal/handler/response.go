package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// errorResponse is the JSON body for all error responses.
type errorResponse struct {
	Error string `json:"error"`
}

// tokenResponse is the JSON body returned upon successful login or refresh.
type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// writeError writes a JSON error body with the given HTTP status.
// Generic messages are preferred for security-sensitive responses (no enumeration).
func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}

// writeTokens writes a successful token pair response.
func writeTokens(c *gin.Context, accessToken, refreshToken string, expiresIn int64) {
	c.JSON(http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}
