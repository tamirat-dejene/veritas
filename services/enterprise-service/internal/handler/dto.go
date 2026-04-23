package handler

import (
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// EnterpriseRegisterRequest represents the request body for enterprise registration.
type EnterpriseRegisterRequest struct {
	Slug          string `json:"slug" binding:"required"`
	DisplayName   string `json:"displayName" binding:"required"`
	LegalName     string `json:"legalName" binding:"required"`
	ContactEmail  string `json:"contactEmail" binding:"required,email"`
	OwnerEmail    string `json:"ownerEmail" binding:"required,email"`
	OwnerPassword string `json:"ownerPassword" binding:"required,min=8"`
}

// EnterpriseListResponse represents a paginated list of enterprises.
type EnterpriseListResponse struct {
	Items []*domain.Enterprise `json:"items"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// UserListResponse represents a paginated list of enterprise users.
type UserListResponse struct {
	Items []*domain.User `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// AuditLogListResponse represents a paginated list of audit logs.
type AuditLogListResponse struct {
	Items []*domain.AuditLog `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// ResetPasswordResponse represents the response body containing a temporary password.
type ResetPasswordResponse struct {
	TemporaryPassword string `json:"temporary_password"`
}
