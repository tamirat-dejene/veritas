package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/google/uuid"
	"github.com/tamirat-dejene/veritas/services/notification-service/internal/domain"
	emailtemplate "github.com/tamirat-dejene/veritas/services/notification-service/internal/template"
	"go.uber.org/zap"
)

type notificationUsecase struct {
	mailer               domain.Mailer
	logger               *zap.Logger
	welcomeTemplate      *template.Template
	passwordResetTemplate *template.Template
}

// EnterpriseStaffCreatedEvent matches the payload sent by enterprise-service
type EnterpriseStaffCreatedEvent struct {
	StaffID        uuid.UUID `json:"staff_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	TempPassword   string    `json:"temp_password"`
	EnterpriseName string    `json:"enterprise_name"`
	Timestamp      int64     `json:"timestamp"`
}

// NewNotificationUsecase creates a new NotificationUsecase.
func NewNotificationUsecase(mailer domain.Mailer, logger *zap.Logger) (domain.NotificationUsecase, error) {
	// Parse the welcome email template
	welcomeTmpl, err := template.New("welcome_staff_email.html").Parse(emailtemplate.WelcomeStaffEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to parse welcome template: %w", err)
	}

	// Parse the password reset email template
	resetTmpl, err := template.New("password_reset_email.html").Parse(emailtemplate.PasswordResetEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to parse password reset template: %w", err)
	}

	return &notificationUsecase{
		mailer:                mailer,
		logger:                logger,
		welcomeTemplate:       welcomeTmpl,
		passwordResetTemplate: resetTmpl,
	}, nil
}

// HandleEnterpriseStaffCreated handles the enterprise.staff.created event.
func (uc *notificationUsecase) HandleEnterpriseStaffCreated(ctx context.Context, payload []byte) error {
	var event EnterpriseStaffCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseStaffCreatedEvent", zap.Error(err))
		return err
	}

	uc.logger.Info("Processing welcome email", zap.String("email", event.Email))

	// Render the template
	var bodyBuffer bytes.Buffer
	err := uc.welcomeTemplate.Execute(&bodyBuffer, map[string]interface{}{
		"Name":           event.Name,
		"EnterpriseName": event.EnterpriseName,
		"Email":          event.Email,
		"TempPassword":   event.TempPassword,
	})
	if err != nil {
		uc.logger.Error("Failed to render welcome email template", zap.Error(err))
		return err
	}

	req := domain.EmailRequest{
		To:      event.Email,
		Subject: fmt.Sprintf("Welcome to %s!", event.EnterpriseName),
		Body:    bodyBuffer.String(),
	}

	// Send the email
	if err := uc.mailer.SendEmail(ctx, req); err != nil {
		uc.logger.Error("Failed to send welcome email", zap.Error(err))
		return err
	}

	uc.logger.Info("Successfully sent welcome email", zap.String("email", event.Email))
	return nil
}

// PasswordResetRequestedEvent matches the payload sent by enterprise-service
type PasswordResetRequestedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	ResetLink string `json:"reset_link"`
	Timestamp int64  `json:"timestamp"`
}

// HandlePasswordResetRequested handles the enterprise.password.reset.requested event.
func (uc *notificationUsecase) HandlePasswordResetRequested(ctx context.Context, payload []byte) error {
	var event PasswordResetRequestedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal PasswordResetRequestedEvent", zap.Error(err))
		return err
	}

	uc.logger.Info("Processing password reset email", zap.String("email", event.Email))

	var bodyBuffer bytes.Buffer
	err := uc.passwordResetTemplate.Execute(&bodyBuffer, map[string]interface{}{
		"Name":      event.Name,
		"ResetLink": event.ResetLink,
	})
	if err != nil {
		uc.logger.Error("Failed to render password reset email template", zap.Error(err))
		return err
	}

	req := domain.EmailRequest{
		To:      event.Email,
		Subject: "Reset your Veritas password",
		Body:    bodyBuffer.String(),
	}

	if err := uc.mailer.SendEmail(ctx, req); err != nil {
		uc.logger.Error("Failed to send password reset email", zap.Error(err))
		return err
	}

	uc.logger.Info("Successfully sent password reset email", zap.String("email", event.Email))
	return nil
}
