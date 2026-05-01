package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

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

	examCreatedAdminTmpl      *template.Template
	examScheduledAdminTmpl    *template.Template
	examScheduledCandidateTmpl *template.Template
	examPublishedAdminTmpl    *template.Template
	examPublishedCandidateTmpl *template.Template
	examClosedAdminTmpl       *template.Template
	examClosedCandidateTmpl   *template.Template
	candidateInvitationTmpl   *template.Template
	examSubmittedConfirmationTmpl *template.Template
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

	exCreAdm, _ := parse("exam_created_admin.html", emailtemplate.ExamCreatedAdminEmail)
	exSchAdm, _ := parse("exam_scheduled_admin.html", emailtemplate.ExamScheduledAdminEmail)
	exSchCan, _ := parse("exam_scheduled_candidate.html", emailtemplate.ExamScheduledCandidateEmail)
	exPubAdm, _ := parse("exam_published_admin.html", emailtemplate.ExamPublishedAdminEmail)
	exPubCan, _ := parse("exam_published_candidate.html", emailtemplate.ExamPublishedCandidateEmail)
	exCloAdm, _ := parse("exam_closed_admin.html", emailtemplate.ExamClosedAdminEmail)
	exCloCan, _ := parse("exam_closed_candidate.html", emailtemplate.ExamClosedCandidateEmail)
	candInv, _ := parse("candidate_invitation_email.html", emailtemplate.CandidateInvitationEmail)
	examSubConf, _ := parse("exam_submitted_confirmation.html", emailtemplate.ExamSubmittedConfirmationEmail)

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

		examCreatedAdminTmpl:      exCreAdm,
		examScheduledAdminTmpl:    exSchAdm,
		examScheduledCandidateTmpl: exSchCan,
		examPublishedAdminTmpl:    exPubAdm,
		examPublishedCandidateTmpl: exPubCan,
		examClosedAdminTmpl:       exCloAdm,
		examClosedCandidateTmpl:   exCloCan,
		candidateInvitationTmpl:   candInv,
		examSubmittedConfirmationTmpl: examSubConf,
	}, nil
}

// ---- Enterprise Event Handlers ----

type EnterpriseStaffCreatedEvent struct {
	StaffID        uuid.UUID `json:"staff_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	TempPassword   string    `json:"temp_password"`
	EnterpriseName string    `json:"enterprise_name"`
	Timestamp      int64     `json:"timestamp"`
}

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

type PasswordResetRequestedEvent struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	ResetLink string `json:"reset_link"`
	Timestamp int64  `json:"timestamp"`
}

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

// ---- Exam Event Handlers ----

type ExamCreatedEvent struct {
	ExamID       uuid.UUID `json:"exam_id"`
	EnterpriseID uuid.UUID `json:"enterprise_id"`
	Title        string    `json:"title"`
	AdminEmail   string    `json:"admin_email"`
	Timestamp    int64     `json:"timestamp"`
}

type ExamLifecycleEvent struct {
	ExamID          uuid.UUID  `json:"exam_id"`
	EnterpriseID    uuid.UUID  `json:"enterprise_id"`
	Title           string     `json:"title"`
	AdminEmail      string     `json:"admin_email"`
	CandidateEmails []string   `json:"candidate_emails"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Timestamp       int64      `json:"timestamp"`
}

func (uc *notificationUsecase) HandleExamCreated(ctx context.Context, payload []byte) error {
	var event ExamCreatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal ExamCreatedEvent", zap.Error(err))
		return err
	}

	if event.AdminEmail != "" {
		return uc.sendLifecycleEmail(ctx, event.AdminEmail, "Exam Created", uc.examCreatedAdminTmpl, map[string]interface{}{
			"Title": event.Title,
		})
	}
	return nil
}

func (uc *notificationUsecase) HandleExamScheduled(ctx context.Context, payload []byte) error {
	var event ExamLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal ExamLifecycleEvent", zap.Error(err))
		return err
	}

	data := map[string]interface{}{
		"Title":     event.Title,
		"StartTime": event.StartTime,
		"EndTime":   event.EndTime,
	}

	// Notify Admin
	if event.AdminEmail != "" {
		_ = uc.sendLifecycleEmail(ctx, event.AdminEmail, "Exam Scheduled", uc.examScheduledAdminTmpl, data)
	}

	// Notify Candidates
	for _, email := range event.CandidateEmails {
		email := email // shadow for goroutine
		go func() {
			_ = uc.sendLifecycleEmail(context.Background(), email, "Upcoming Exam", uc.examScheduledCandidateTmpl, data)
		}()
	}

	return nil
}

func (uc *notificationUsecase) HandleExamPublished(ctx context.Context, payload []byte) error {
	var event ExamLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal ExamLifecycleEvent", zap.Error(err))
		return err
	}

	data := map[string]interface{}{"Title": event.Title}

	// Notify Admin
	if event.AdminEmail != "" {
		_ = uc.sendLifecycleEmail(ctx, event.AdminEmail, "Exam Published", uc.examPublishedAdminTmpl, data)
	}

	// Notify Candidates
	for _, email := range event.CandidateEmails {
		email := email
		go func() {
			_ = uc.sendLifecycleEmail(context.Background(), email, "Exam Now Live", uc.examPublishedCandidateTmpl, data)
		}()
	}

	return nil
}

func (uc *notificationUsecase) HandleExamClosed(ctx context.Context, payload []byte) error {
	var event ExamLifecycleEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal ExamLifecycleEvent", zap.Error(err))
		return err
	}

	data := map[string]interface{}{"Title": event.Title}

	// Notify Admin
	if event.AdminEmail != "" {
		_ = uc.sendLifecycleEmail(ctx, event.AdminEmail, "Exam Closed", uc.examClosedAdminTmpl, data)
	}

	// Notify Candidates
	for _, email := range event.CandidateEmails {
		go func() {
			_ = uc.sendLifecycleEmail(context.Background(), email, "Exam Finished", uc.examClosedCandidateTmpl, data)
		}()
	}

	return nil
}

type CandidateEnrollmentInvitedEvent struct {
	EnrollmentID   uuid.UUID `json:"enrollment_id"`
	CandidateID    uuid.UUID `json:"candidate_id"`
	ExamID         uuid.UUID `json:"exam_id"`
	EnterpriseID   uuid.UUID `json:"enterprise_id"`
	CandidateName  string    `json:"candidate_name"`
	CandidateEmail string    `json:"candidate_email"`
	ExamTitle      string    `json:"exam_title"`
	InvitationURL  string    `json:"invitation_url"`
	ExpiresAt      time.Time `json:"expires_at"`
	Timestamp      int64     `json:"timestamp"`
}

func (uc *notificationUsecase) HandleCandidateEnrollmentInvited(ctx context.Context, payload []byte) error {
	var event CandidateEnrollmentInvitedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal CandidateEnrollmentInvitedEvent", zap.Error(err))
		return err
	}

	uc.logger.Info("Processing candidate invitation email", zap.String("email", event.CandidateEmail))

	var bodyBuffer bytes.Buffer
	err := uc.candidateInvitationTmpl.Execute(&bodyBuffer, map[string]interface{}{
		"Name":          event.CandidateName,
		"ExamTitle":     event.ExamTitle,
		"InvitationURL": event.InvitationURL,
		"ExpiresAt":     event.ExpiresAt.Format("Jan 02, 2006 15:04 MST"),
		"Year":          time.Now().Year(),
	})
	if err != nil {
		uc.logger.Error("Failed to render candidate invitation email template", zap.Error(err))
		return err
	}

	req := domain.EmailRequest{
		To:      event.CandidateEmail,
		Subject: fmt.Sprintf("Invitation to take exam: %s", event.ExamTitle),
		Body:    bodyBuffer.String(),
	}

	if err := uc.mailer.SendEmail(ctx, req); err != nil {
		uc.logger.Error("Failed to send candidate invitation email", zap.Error(err))
		return err
	}

	uc.logger.Info("Successfully sent candidate invitation email", zap.String("email", event.CandidateEmail))
	return nil
}

type CandidateExamSubmittedEvent struct {
	SessionID      uuid.UUID `json:"session_id"`
	CandidateID    uuid.UUID `json:"candidate_id"`
	ExamID         uuid.UUID `json:"exam_id"`
	EnterpriseID   uuid.UUID `json:"enterprise_id"`
	CandidateName  string    `json:"candidate_name"`
	CandidateEmail string    `json:"candidate_email"`
	ExamTitle      string    `json:"exam_title"`
	SubmittedAt    time.Time `json:"submitted_at"`
	AutoSubmitted  bool      `json:"auto_submitted"`
	Timestamp      int64     `json:"timestamp"`
}

func (uc *notificationUsecase) HandleCandidateExamSubmitted(ctx context.Context, payload []byte) error {
	var event CandidateExamSubmittedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		uc.logger.Error("Failed to unmarshal CandidateExamSubmittedEvent", zap.Error(err))
		return err
	}

	uc.logger.Info("Processing exam submitted confirmation email", zap.String("email", event.CandidateEmail))

	var bodyBuffer bytes.Buffer
	err := uc.examSubmittedConfirmationTmpl.Execute(&bodyBuffer, map[string]interface{}{
		"CandidateName": event.CandidateName,
		"ExamTitle":     event.ExamTitle,
		"SubmittedAt":   event.SubmittedAt.Format("Jan 02, 2006 15:04 MST"),
		"AutoSubmitted": event.AutoSubmitted,
	})
	if err != nil {
		uc.logger.Error("Failed to render exam submitted confirmation template", zap.Error(err))
		return err
	}

	req := domain.EmailRequest{
		To:      event.CandidateEmail,
		Subject: fmt.Sprintf("Exam Submitted: %s", event.ExamTitle),
		Body:    bodyBuffer.String(),
	}

	if err := uc.mailer.SendEmail(ctx, req); err != nil {
		uc.logger.Error("Failed to send exam submitted confirmation email", zap.Error(err))
		return err
	}

	uc.logger.Info("Successfully sent exam submitted confirmation email", zap.String("email", event.CandidateEmail))
	return nil
}

