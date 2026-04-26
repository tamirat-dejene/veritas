package emailtemplate

import (
	_ "embed"
)

//go:embed welcome_staff_email.html
var WelcomeStaffEmail string

//go:embed password_reset_email.html
var PasswordResetEmail string

//go:embed enterprise_registered_email.html
var EnterpriseRegisteredEmail string

//go:embed enterprise_approved_email.html
var EnterpriseApprovedEmail string

//go:embed enterprise_suspended_email.html
var EnterpriseSuspendedEmail string

//go:embed enterprise_deleted_email.html
var EnterpriseDeletedEmail string

//go:embed enterprise_hard_deleted_email.html
var EnterpriseHardDeletedEmail string

//go:embed enterprise_reactivated_email.html
var EnterpriseReactivatedEmail string

//go:embed enterprise_restored_email.html
var EnterpriseRestoredEmail string

//go:embed user_deactivated_email.html
var UserDeactivatedEmail string

//go:embed user_activated_email.html
var UserActivatedEmail string

//go:embed user_password_changed_email.html
var UserPasswordChangedEmail string

//go:embed user_password_reset_admin_email.html
var UserPasswordResetAdminEmail string

//go:embed exam_created_admin.html
var ExamCreatedAdminEmail string

//go:embed exam_scheduled_admin.html
var ExamScheduledAdminEmail string

//go:embed exam_scheduled_candidate.html
var ExamScheduledCandidateEmail string

//go:embed exam_published_admin.html
var ExamPublishedAdminEmail string

//go:embed exam_published_candidate.html
var ExamPublishedCandidateEmail string

//go:embed exam_closed_admin.html
var ExamClosedAdminEmail string

//go:embed exam_closed_candidate.html
var ExamClosedCandidateEmail string
