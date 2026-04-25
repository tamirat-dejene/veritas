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
}
