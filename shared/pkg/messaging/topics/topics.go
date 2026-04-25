package topics

// Topic names pattern: <service>.<entity>.<action>

// Auth-Service topics
const (
	AuthUserLogin = "auth.user.login"
)

// Enterprise-Service topics
const (
	EnterpriseCreated                = "enterprise.enterprise.created"
	EnterpriseApproved               = "enterprise.enterprise.approved"
	EnterpriseSuspended              = "enterprise.enterprise.suspended"
	EnterpriseDeleted                = "enterprise.enterprise.deleted"
	EnterpriseHardDeleted            = "enterprise.enterprise.hard_deleted"
	EnterpriseReactivated            = "enterprise.enterprise.reactivated"
	EnterpriseRestored               = "enterprise.enterprise.restored"
	EnterpriseStaffCreated           = "enterprise.staff.created"
	EnterprisePasswordResetRequested = "enterprise.password.reset.requested"
	UserDeactivated                  = "enterprise.user.deactivated"
	UserActivated                    = "enterprise.user.activated"
	UserPasswordChanged              = "enterprise.user.password.changed"
	UserPasswordResetAdmin           = "enterprise.user.password.reset.admin"
)

// Payment-Service topics
const (
	SubscriptionPaymentFailed = "subscription.payment.failed"
	SubscriptionUpdated       = "subscription.updated"
	SubscriptionCanceled      = "subscription.canceled"
	InvoiceUpcoming           = "payment.invoice.upcoming"
)