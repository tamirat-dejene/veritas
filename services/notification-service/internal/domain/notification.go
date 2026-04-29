package domain

import (
	"context"
)

// EmailRequest represents the data needed to send an email.
type EmailRequest struct {
	To      string
	Subject string
	Body    string // HTML body
}

// Mailer defines the interface for an email sending service.
type Mailer interface {
	SendEmail(ctx context.Context, req EmailRequest) error
}

// NotificationUsecase handles the business logic for incoming events.
type NotificationUsecase interface {
	HandleEnterpriseStaffCreated(ctx context.Context, payload []byte) error
	HandlePasswordResetRequested(ctx context.Context, payload []byte) error
	HandleEnterpriseCreated(ctx context.Context, payload []byte) error
	HandleEnterpriseApproved(ctx context.Context, payload []byte) error
	HandleEnterpriseSuspended(ctx context.Context, payload []byte) error
	HandleEnterpriseDeleted(ctx context.Context, payload []byte) error
	HandleEnterpriseHardDeleted(ctx context.Context, payload []byte) error
	HandleEnterpriseReactivated(ctx context.Context, payload []byte) error
	HandleEnterpriseRestored(ctx context.Context, payload []byte) error
	HandleUserDeactivated(ctx context.Context, payload []byte) error
	HandleUserActivated(ctx context.Context, payload []byte) error
	HandleUserPasswordChanged(ctx context.Context, payload []byte) error
	HandleUserPasswordResetAdmin(ctx context.Context, payload []byte) error

	HandleExamCreated(ctx context.Context, payload []byte) error
	HandleExamScheduled(ctx context.Context, payload []byte) error
	HandleExamPublished(ctx context.Context, payload []byte) error
	HandleExamClosed(ctx context.Context, payload []byte) error

	HandleCandidateEnrollmentInvited(ctx context.Context, payload []byte) error
}
