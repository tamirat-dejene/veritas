package handler

import (
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/enterprise-service/internal/domain"
)

// ErrorResponse is the standard error payload for this service.
type ErrorResponse struct {
	Error string `json:"error"`
}

// EnterpriseRegisterRequest is the request body for enterprise registration.
type EnterpriseRegisterRequest struct {
	Slug          string `json:"slug" binding:"required"`
	DisplayName   string `json:"displayName" binding:"required"`
	LegalName     string `json:"legalName" binding:"required"`
	ContactEmail  string `json:"contactEmail" binding:"required,email"`
	OwnerEmail    string `json:"ownerEmail" binding:"required,email"`
	OwnerPassword string `json:"ownerPassword" binding:"required,min=8"`
}

// EnterpriseListResponse models paginated enterprise list output.
type EnterpriseListResponse struct {
	Items []*domain.Enterprise `json:"items"`
	Total int                  `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// UserListResponse models paginated enterprise user list output.
type UserListResponse struct {
	Items []*domain.User `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

// AuditLogListResponse models paginated audit log list output.
type AuditLogListResponse struct {
	Items []*domain.AuditLog `json:"items"`
	Total int                `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}

// SubscriptionInfoResponse is returned by the subscription info endpoint.
type SubscriptionInfoResponse struct {
	EnterpriseID       uuid.UUID                  `json:"enterprise_id"`
	SubscriptionPlanID *uuid.UUID                 `json:"subscription_plan_id"`
	SubscriptionStatus *domain.SubscriptionStatus `json:"subscription_status"`
	CurrentPeriodStart *time.Time                 `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time                 `json:"current_period_end"`
}

// ResetPasswordResponse carries a temporary password for user reset operations.
type ResetPasswordResponse struct {
	TemporaryPassword string `json:"temporary_password"`
}
