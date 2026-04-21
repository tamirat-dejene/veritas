package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// EnterpriseFilter holds query parameters for listing enterprises.
type EnterpriseFilter struct {
	pagination.Params
	Status             *EnterpriseStatus
	SubscriptionStatus *SubscriptionStatus
	Search             string // searches display_name and slug
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
	ID   uuid.UUID `db:"id" json:"id"`
	Slug string    `db:"slug" json:"slug"`

	DisplayName  string `db:"display_name" json:"displayName"`
	LegalName    string `db:"legal_name" json:"legalName"`
	ContactEmail string `db:"contact_email" json:"contactEmail"`

	OwnerAccountID uuid.UUID `db:"owner_account_id" json:"ownerAccountId"`

	Status      EnterpriseStatus `db:"status" json:"status"`
	ApprovedAt  *time.Time       `db:"approved_at" json:"approvedAt"`
	SuspendedAt *time.Time       `db:"suspended_at" json:"suspendedAt"`
	DeletedAt   *time.Time       `db:"deleted_at" json:"deletedAt"`

	SubscriptionPlanID *uuid.UUID          `db:"subscription_plan_id" json:"subscriptionPlanId,omitempty"`
	SubscriptionStatus *SubscriptionStatus `db:"subscription_status" json:"subscriptionStatus,omitempty"`
	CurrentPeriodStart *time.Time          `db:"current_period_start" json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time          `db:"current_period_end" json:"currentPeriodEnd,omitempty"`

	LogoURL        *string `db:"logo_url" json:"logoUrl,omitempty"`
	PrimaryColor   *string `db:"primary_color" json:"primaryColor,omitempty"`
	SecondaryColor *string `db:"secondary_color" json:"secondaryColor,omitempty"`
	CustomDomain   *string `db:"custom_domain" json:"customDomain,omitempty"`

	ContactPhone *string `db:"contact_phone" json:"contactPhone,omitempty"`
	AddressLine1 *string `db:"address_line1" json:"addressLine1,omitempty"`
	AddressLine2 *string `db:"address_line2" json:"addressLine2,omitempty"`
	City         *string `db:"city" json:"city,omitempty"`
	Country      *string `db:"country" json:"country,omitempty"`

	Settings map[string]any `db:"settings" json:"settings"`

	RetentionUntil *time.Time `db:"retention_until" json:"retentionUntil,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
	CreatedBy uuid.UUID `db:"created_by" json:"createdBy"`
	UpdatedBy uuid.UUID `db:"updated_by" json:"updatedBy"`
}

// NewEnterprise provides a standard way to initialize an Enterprise.
func NewEnterprise(id uuid.UUID, slug, displayName string, ownerID uuid.UUID) *Enterprise {
	now := time.Now().UTC()
	return &Enterprise{
		ID:             id,
		Slug:           slug,
		DisplayName:    displayName,
		OwnerAccountID: ownerID,
		Status:         StatusPendingApproval,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      ownerID,
		UpdatedBy:      ownerID,
		Settings:       make(map[string]any),
	}
}
