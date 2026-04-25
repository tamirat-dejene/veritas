package mailer

import (
	"context"

	"github.com/tamirat-dejene/veritas/services/notification-service/internal/config"
	"github.com/tamirat-dejene/veritas/services/notification-service/internal/domain"
	"gopkg.in/gomail.v2"
)

type smtpMailer struct {
	cfg *config.Config
}

// NewSMTPMailer creates a new SMTP mailer instance.
func NewSMTPMailer(cfg *config.Config) domain.Mailer {
	return &smtpMailer{
		cfg: cfg,
	}
}

// SendEmail sends an email using the configured SMTP server.
func (m *smtpMailer) SendEmail(ctx context.Context, req domain.EmailRequest) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.SMTPFrom)
	msg.SetHeader("To", req.To)
	msg.SetHeader("Subject", req.Subject)
	msg.SetBody("text/html", req.Body)

	// If SMTPAuth is required, provide the user and pass
	dialer := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUser, m.cfg.SMTPPass)

	// If we're using a mock like Mailhog, it usually doesn't have TLS/Auth
	// Gomail usually handles no-auth if the credentials are empty strings.

	if err := dialer.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
