package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the JSON body for all error responses.
type ErrorResponse struct {
	Code      string `json:"code" example:"invalid_credentials"`
	Message   string `json:"message" example:"invalid email or password"`
	RequestID string `json:"requestId,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// BadRequestErrorResponse documents the 400 error payload shape and examples.
type BadRequestErrorResponse struct {
	Code      string `json:"code" example:"invalid_request"`
	Message   string `json:"message" example:"invalid request body"`
	RequestID string `json:"requestId,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// UnauthorizedErrorResponse documents the 401 error payload shape and examples.
type UnauthorizedErrorResponse struct {
	Code      string `json:"code" example:"invalid_credentials"`
	Message   string `json:"message" example:"invalid email or password"`
	RequestID string `json:"requestId,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// ForbiddenErrorResponse documents the 403 error payload shape and examples.
type ForbiddenErrorResponse struct {
	Code      string `json:"code" example:"access_denied"`
	Message   string `json:"message" example:"access denied"`
	RequestID string `json:"requestId,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// InternalErrorResponse documents the 500 error payload shape and examples.
type InternalErrorResponse struct {
	Code      string `json:"code" example:"internal_error"`
	Message   string `json:"message" example:"internal server error"`
	RequestID string `json:"requestId,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
}

// TokenResponse is the JSON body returned upon successful login or refresh.
type TokenResponse struct {
	AccessToken  string `json:"accessToken"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refreshToken" example:"b8a54f0f0cc6d2f68dd0b457ea4bb7f814ff69ec487f474f5c6f1781b6f0a0d3"`
	ExpiresIn    int64  `json:"expiresIn"    example:"3600"`
}

// writeError writes a JSON error body with the given HTTP status.
// Generic messages are preferred for security-sensitive responses (no enumeration).
func writeError(c *gin.Context, status int, code, message string) {
	requestID, ok := c.Get("requestId")
	requestIDStr, _ := requestID.(string)
	if !ok {
		requestIDStr = ""
	}

	c.JSON(status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: requestIDStr,
	})
}

// writeTokens writes a successful token pair response.
func writeTokens(c *gin.Context, accessToken, refreshToken string, expiresIn int64) {
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})
}
