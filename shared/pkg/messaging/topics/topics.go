package topics

// Topic names pattern: <service>.<entity>.<action>

// Auth-Service topics
const (
	AuthUserLogin = "auth.user.login"
)

// Enterprise-Service topics
const (
	EnterpriseCreated = "enterprise.enterprise.created"
)

// Payment-Service topics
const (
	SubscriptionPaymentFailed = "subscription.payment.failed"
	SubscriptionUpdated       = "subscription.updated"
	SubscriptionCanceled      = "subscription.canceled"
	InvoiceUpcoming           = "payment.invoice.upcoming"
)