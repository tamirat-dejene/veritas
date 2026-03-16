package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent is the type of action that was performed.
type AuditEvent string

const (
	EventEnterpriseCreated     AuditEvent = "enterprise.created"
	EventEnterpriseUpdated     AuditEvent = "enterprise.updated"
	EventEnterpriseApproved    AuditEvent = "enterprise.approved"
	EventEnterpriseSuspended   AuditEvent = "enterprise.suspended"
	EventEnterpriseDeleted     AuditEvent = "enterprise.deleted"
	EventEnterpriseReactivated AuditEvent = "enterprise.reactivated"
	EventEnterpriseRestored    AuditEvent = "enterprise.restored"
	EventEnterpriseHardDeleted AuditEvent = "enterprise.hard_deleted"
	EventBrandingUpdated       AuditEvent = "enterprise.branding_updated"
	EventSettingsUpdated       AuditEvent = "enterprise.settings_updated"
	EventSubscriptionUpdated   AuditEvent = "subscription.updated"
	EventSubscriptionCanceled  AuditEvent = "subscription.canceled"
	EventSubscriptionRenewed   AuditEvent = "subscription.renewed"
	EventSubscriptionSuspended AuditEvent = "subscription.payment_suspended"
	EventUserCreated           AuditEvent = "user.created"
	EventUserUpdated           AuditEvent = "user.updated"
	EventUserDeactivated       AuditEvent = "user.deactivated"
	EventUserPasswordReset     AuditEvent = "user.password_reset"
	EventDomainValidated       AuditEvent = "enterprise.domain_validated"
)

// AuditLog records a single auditable action on an enterprise.
type AuditLog struct {
	ID           uuid.UUID              `db:"id"            json:"id"`
	EnterpriseID uuid.UUID              `db:"enterprise_id" json:"enterprise_id"`
	ActorID      uuid.UUID              `db:"actor_id"      json:"actor_id"`
	ActorRole    string                 `db:"actor_role"    json:"actor_role"`
	Event        AuditEvent             `db:"event"         json:"event"`
	Metadata     map[string]any 		`db:"metadata"      json:"metadata"`
	CreatedAt    time.Time              `db:"created_at"    json:"created_at"`
}
