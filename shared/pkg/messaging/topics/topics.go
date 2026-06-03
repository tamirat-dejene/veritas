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
	UserDeleted                      = "enterprise.user.deleted"
)

// Payment-Service topics
const (
	SubscriptionPaymentFailed = "subscription.payment.failed"
	SubscriptionUpdated       = "subscription.updated"
	SubscriptionCanceled      = "subscription.canceled"
	InvoiceUpcoming           = "payment.invoice.upcoming"
)

// Exam-Service topics
const (
	ExamCreated   = "exam.exam.created"
	ExamScheduled = "exam.exam.scheduled"
	ExamPublished = "exam.exam.published"
	ExamClosed    = "exam.exam.closed"
)

// Candidate-Service topics
const (
	// CandidateEnrollmentInvited is published when an enterprise admin triggers
	// notification for one or more candidates. The notification-service consumes
	// this topic and sends the branded invitation email.
	CandidateEnrollmentInvited = "candidate.enrollment.invited"

	// CandidateExamSubmitted is published when a candidate submits their exam session.
	// The notification-service consumes this topic and sends a confirmation email.
	CandidateExamSubmitted = "candidate.exam.submitted"

	// ExamSessionReadyForGrading is published when a session is submitted or terminated.
	// The grading-service consumes this slim trigger (v3.0) and pulls the full grading
	// payload via an internal HTTP call to the candidate-service.
	ExamSessionReadyForGrading = "exam.session.ready_for_grading"
)

// Proctoring-Service topics
const (
	// ProctoringIdentityVerified is published after each periodic face verification check.
	// Contains is_match flag, confidence score, and session context.
	ProctoringIdentityVerified = "proctoring.identity.verified"

	// ProctoringEventDetected is published for every behavioral event ingested
	// (tab switch, mouse inactivity, face anomaly, etc.).
	ProctoringEventDetected = "proctoring.event.detected"

	// ProctoringCheatingScore is published whenever the cheating probability score
	// is recomputed for a session. The is_final flag is set to true once the session
	// is submitted, signalling the definitive score to reporting and grading services.
	ProctoringCheatingScore = "proctoring.cheating_score.updated"
)

// Grading-Service topics
const (
	// GradingSessionCompleted is published when an exam session has been graded.
	// The notification-service consumes this to email candidates their results.
	GradingSessionCompleted = "grading.session.completed"
)

