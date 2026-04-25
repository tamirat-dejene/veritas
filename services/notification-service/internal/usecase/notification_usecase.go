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
	enterpriseRegisteredTmpl  *template.Template
	enterpriseApprovedTmpl    *template.Template
	enterpriseSuspendedTmpl   *template.Template
	enterpriseDeletedTmpl     *template.Template
	enterpriseHardDeletedTmpl *template.Template
	enterpriseReactivatedTmpl *template.Template
	enterpriseRestoredTmpl    *template.Template
	userDeactivatedTmpl       *template.Template
	userActivatedTmpl         *template.Template
	userPasswordChangedTmpl   *template.Template
	userPasswordResetAdminTmpl *template.Template
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

	parse := func(name, content string) (*template.Template, error) {
		t, err := template.New(name).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", name, err)
		}
		return t, nil
	}

	entRegTmpl, _ := parse("enterprise_registered_email.html", emailtemplate.EnterpriseRegisteredEmail)
	entAppTmpl, _ := parse("enterprise_approved_email.html", emailtemplate.EnterpriseApprovedEmail)
	entSusTmpl, _ := parse("enterprise_suspended_email.html", emailtemplate.EnterpriseSuspendedEmail)
	entDelTmpl, _ := parse("enterprise_deleted_email.html", emailtemplate.EnterpriseDeletedEmail)
	entHDelTmpl, _ := parse("enterprise_hard_deleted_email.html", emailtemplate.EnterpriseHardDeletedEmail)
	entReaTmpl, _ := parse("enterprise_reactivated_email.html", emailtemplate.EnterpriseReactivatedEmail)
	entResTmpl, _ := parse("enterprise_restored_email.html", emailtemplate.EnterpriseRestoredEmail)

	userDeacTmpl, _ := parse("user_deactivated_email.html", emailtemplate.UserDeactivatedEmail)
	userActTmpl, _ := parse("user_activated_email.html", emailtemplate.UserActivatedEmail)
	userPwdChgTmpl, _ := parse("user_password_changed_email.html", emailtemplate.UserPasswordChangedEmail)
	userPwdRstAdmTmpl, _ := parse("user_password_reset_admin_email.html", emailtemplate.UserPasswordResetAdminEmail)

	return &notificationUsecase{
		mailer:                mailer,
		logger:                logger,
		welcomeTemplate:       welcomeTmpl,
		passwordResetTemplate: resetTmpl,
		enterpriseRegisteredTmpl:  entRegTmpl,
		enterpriseApprovedTmpl:    entAppTmpl,
		enterpriseSuspendedTmpl:   entSusTmpl,
		enterpriseDeletedTmpl:     entDelTmpl,
		enterpriseHardDeletedTmpl: entHDelTmpl,
		enterpriseReactivatedTmpl: entReaTmpl,
		enterpriseRestoredTmpl:    entResTmpl,
		userDeactivatedTmpl:       userDeacTmpl,
		userActivatedTmpl:         userActTmpl,
		userPasswordChangedTmpl:   userPwdChgTmpl,
		userPasswordResetAdminTmpl: userPwdRstAdmTmpl,
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

type EnterpriseCreatedEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	OwnerEmail   string    `json:"owner_email"`
	Timestamp    int64     `json:"timestamp"`
}

type EnterpriseLifecycleEvent struct {
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

func (uc *notificationUsecase) HandleEnterpriseCreated(ctx context.Context, payload []byte) error {
	var event EnterpriseCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseCreatedEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.OwnerEmail, "Registration Complete", uc.enterpriseRegisteredTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) HandleEnterpriseApproved(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Approved", uc.enterpriseApprovedTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) HandleEnterpriseSuspended(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Suspended", uc.enterpriseSuspendedTmpl, map[string]interface{}{
		"Name":   event.Name,
		"Reason": event.Reason,
	})
}

func (uc *notificationUsecase) HandleEnterpriseDeleted(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Deletion Initiated", uc.enterpriseDeletedTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) HandleEnterpriseHardDeleted(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Permanently Deleted", uc.enterpriseHardDeletedTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) HandleEnterpriseReactivated(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Reactivated", uc.enterpriseReactivatedTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) HandleEnterpriseRestored(ctx context.Context, payload []byte) error {
	var event EnterpriseLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal EnterpriseLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.ContactEmail, "Enterprise Restored", uc.enterpriseRestoredTmpl, map[string]interface{}{
		"Name": event.Name,
	})
}

func (uc *notificationUsecase) sendLifecycleEmail(ctx context.Context, to, subject string, tmpl *template.Template, data map[string]interface{}) error {
	var bodyBuffer bytes.Buffer
	if err := tmpl.Execute(&bodyBuffer, data); err != nil {
		uc.logger.Error("Failed to render lifecycle template", zap.Error(err))
		return err
	}

	req := domain.EmailRequest{
		To:      to,
		Subject: subject,
		Body:    bodyBuffer.String(),
	}

	if err := uc.mailer.SendEmail(ctx, req); err != nil {
		uc.logger.Error("Failed to send lifecycle email", zap.Error(err), zap.String("subject", subject))
		return err
	}
	return nil
}

type UserLifecycleEvent struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	EnterpriseName string    `json:"enterprise_name"`
	TempPassword   string    `json:"temp_password,omitempty"`
	Timestamp      int64     `json:"timestamp"`
}

func (uc *notificationUsecase) HandleUserDeactivated(ctx context.Context, payload []byte) error {
	var event UserLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal UserLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.Email, "Account Deactivated", uc.userDeactivatedTmpl, map[string]interface{}{
		"Name":           event.Name,
		"EnterpriseName": event.EnterpriseName,
	})
}

func (uc *notificationUsecase) HandleUserActivated(ctx context.Context, payload []byte) error {
	var event UserLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal UserLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.Email, "Account Activated", uc.userActivatedTmpl, map[string]interface{}{
		"Name":           event.Name,
		"EnterpriseName": event.EnterpriseName,
	})
}

func (uc *notificationUsecase) HandleUserPasswordChanged(ctx context.Context, payload []byte) error {
	var event UserLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal UserLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.Email, "Password Changed", uc.userPasswordChangedTmpl, map[string]interface{}{
		"Name":           event.Name,
		"EnterpriseName": event.EnterpriseName,
	})
}

func (uc *notificationUsecase) HandleUserPasswordResetAdmin(ctx context.Context, payload []byte) error {
	var event UserLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal UserLifecycleEvent", zap.Error(err))
		return err
	}
	return uc.sendLifecycleEmail(ctx, event.Email, "Password Reset", uc.userPasswordResetAdminTmpl, map[string]interface{}{
		"Name":           event.Name,
		"EnterpriseName": event.EnterpriseName,
		"TempPassword":   event.TempPassword,
	})
}
