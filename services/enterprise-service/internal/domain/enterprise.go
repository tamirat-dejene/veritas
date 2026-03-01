package domain

import (
	"time"

	"github.com/google/uuid"
)

// EnterpriseFilter holds query parameters for listing enterprises.
type EnterpriseFilter struct {
	Status             *EnterpriseStatus
	SubscriptionStatus *SubscriptionStatus
	Search             string // searches display_name and slug
	Page               int
	Limit              int
}

// PaginatedResult is a generic paginated response wrapper.
type PaginatedResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// UpdateBrandingRequest carries the allowed branding fields.
type UpdateBrandingRequest struct {
	LogoURL        *string `json:"logo_url"`
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
}

// UpdateSubscriptionRequest carries subscription update fields.
type UpdateSubscriptionRequest struct {
	SubscriptionPlanID *uuid.UUID          `json:"subscription_plan_id"`
	SubscriptionStatus *SubscriptionStatus `json:"subscription_status"`
	PeriodStart        *time.Time          `json:"period_start"`
	PeriodEnd          *time.Time          `json:"period_end"`
}

// EnterpriseSummary provides a high-level overview of an enterprise.
type EnterpriseSummary struct {
	EnterpriseID       uuid.UUID           `json:"enterprise_id"`
	DisplayName        string              `json:"display_name"`
	Status             EnterpriseStatus    `json:"status"`
	SubscriptionStatus *SubscriptionStatus `json:"subscription_status"`
	SubscriptionExpiry *time.Time          `json:"subscription_expiry"`
	UserCount          int                 `json:"user_count"`
	// ActiveExamCount and ActiveSessionCount require inter-service calls – set to -1 when unavailable.
	ActiveExamCount    int `json:"active_exam_count"`
	ActiveSessionCount int `json:"active_session_count"`
}

// EnterpriseStatusResponse holds lifecycle + compliance data.
type EnterpriseStatusResponse struct {
	EnterpriseID       uuid.UUID           `json:"enterprise_id"`
	Status             EnterpriseStatus    `json:"status"`
	SubscriptionStatus *SubscriptionStatus `json:"subscription_status"`
	ApprovedAt         *time.Time          `json:"approved_at"`
	SuspendedAt        *time.Time          `json:"suspended_at"`
	DeletedAt          *time.Time          `json:"deleted_at"`
	RetentionUntil     *time.Time          `json:"retention_until"`
	CurrentPeriodEnd   *time.Time          `json:"current_period_end"`
}

type EnterpriseStatus string

const (
	StatusPendingApproval EnterpriseStatus = "PendingApproval"
	StatusActive          EnterpriseStatus = "Active"
	StatusSuspended       EnterpriseStatus = "Suspended"
	StatusDeleted         EnterpriseStatus = "Deleted"
)

type SubscriptionStatus string

const (
	SubTrial    SubscriptionStatus = "Trial"
	SubActive   SubscriptionStatus = "Active"
	SubPastDue  SubscriptionStatus = "PastDue"
	SubCanceled SubscriptionStatus = "Canceled"
	SubExpired  SubscriptionStatus = "Expired"
)

type Enterprise struct {
	ID   uuid.UUID `db:"id"`
	Slug string    `db:"slug"`

	DisplayName  string `db:"display_name"`
	LegalName    string `db:"legal_name"`
	ContactEmail string `db:"contact_email"`

	OwnerAccountID uuid.UUID `db:"owner_account_id"`

	Status      EnterpriseStatus `db:"status"`
	ApprovedAt  *time.Time       `db:"approved_at"`
	SuspendedAt *time.Time       `db:"suspended_at"`
	DeletedAt   *time.Time       `db:"deleted_at"`

	SubscriptionPlanID *uuid.UUID          `db:"subscription_plan_id"`
	SubscriptionStatus *SubscriptionStatus `db:"subscription_status"`
	CurrentPeriodStart *time.Time          `db:"current_period_start"`
	CurrentPeriodEnd   *time.Time          `db:"current_period_end"`

	LogoURL        *string `db:"logo_url"`
	PrimaryColor   *string `db:"primary_color"`
	SecondaryColor *string `db:"secondary_color"`
	CustomDomain   *string `db:"custom_domain"`

	ContactPhone *string `db:"contact_phone"`
	AddressLine1 *string `db:"address_line1"`
	AddressLine2 *string `db:"address_line2"`
	City         *string `db:"city"`
	Country      *string `db:"country"`

	Settings map[string]interface{} `db:"settings"`

	RetentionUntil *time.Time `db:"retention_until"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedBy uuid.UUID `db:"created_by"`
	UpdatedBy uuid.UUID `db:"updated_by"`
}
