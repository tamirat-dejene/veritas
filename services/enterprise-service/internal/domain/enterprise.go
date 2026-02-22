package domain

import (
	"time"

	"github.com/google/uuid"
)

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

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedBy uuid.UUID `db:"created_by"`
	UpdatedBy uuid.UUID `db:"updated_by"`
}
