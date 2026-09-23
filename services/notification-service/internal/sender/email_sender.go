package sender

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/config"
)

type EmailSender interface {
	Send(ctx context.Context, to string, subject string, htmlBody string) error
}

type smtpEmailSender struct {
	cfg *config.Config
}

func NewEmailSender(cfg *config.Config) EmailSender {
	return &smtpEmailSender{cfg: cfg}
}

func (s *smtpEmailSender) Send(ctx context.Context, to string, subject string, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	from := s.cfg.SMTPFromEmail
	fromName := s.cfg.SMTPFromName
	if fromName == "" {
		fromName = "PMB Notification"
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	var auth smtp.Auth
	if s.cfg.SMTPUser != "" && s.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}

	// Support context cancellation check
	if err := ctx.Err(); err != nil {
		return err
	}

	toAddresses := strings.Split(to, ",")
	return smtp.SendMail(addr, auth, from, toAddresses, []byte(message))
}
