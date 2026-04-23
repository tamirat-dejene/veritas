package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/shared/pkg/pagination"
)

// EnterpriseFilter holds query parameters for listing enterprises.
type EnterpriseFilter struct {
	pagination.Params
	Status *EnterpriseStatus
	Search string
}

// UpdateBrandingRequest carries the allowed branding fields.
type UpdateBrandingRequest struct {
	LogoURL        *string `json:"logo_url"`
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
}

// SubscriptionSnapshot is the read-only view of subscription state
type SubscriptionSnapshot struct {
	PlanID             uuid.UUID `json:"plan_id"`
	PlanName           string    `json:"plan_name,omitempty"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
}


// EnterpriseSummary provides a high-level overview of an enterprise.
type EnterpriseSummary struct {
	EnterpriseID       uuid.UUID             `json:"enterprise_id"`
	DisplayName        string                `json:"display_name"`
	Status             EnterpriseStatus      `json:"status"`
	Subscription       *SubscriptionSnapshot `json:"subscription,omitempty"`
	UserCount          int                   `json:"user_count"`
	ActiveExamCount    int                   `json:"active_exam_count"`
	ActiveSessionCount int                   `json:"active_session_count"`
}

// EnterpriseStatusResponse holds lifecycle + compliance data.
type EnterpriseStatusResponse struct {
	EnterpriseID   uuid.UUID             `json:"enterprise_id"`
	Status         EnterpriseStatus      `json:"status"`
	ApprovedAt     *time.Time            `json:"approved_at"`
	SuspendedAt    *time.Time            `json:"suspended_at"`
	DeletedAt      *time.Time            `json:"deleted_at"`
	RetentionUntil *time.Time            `json:"retention_until"`
	Subscription   *SubscriptionSnapshot `json:"subscription,omitempty"`
}

type EnterpriseStatus string

const (
	StatusPendingApproval EnterpriseStatus = "PendingApproval"
	StatusActive          EnterpriseStatus = "Active"
	StatusSuspended       EnterpriseStatus = "Suspended"
	StatusDeleted         EnterpriseStatus = "Deleted"
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
