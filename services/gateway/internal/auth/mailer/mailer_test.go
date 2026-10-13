package mailer

import (
	"context"
	"testing"

	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	"go.uber.org/zap"
)

func TestLogMailer(t *testing.T) {
	log := zap.NewNop()
	m := &logMailer{
		appName: "PMB Universitas",
		log:     log,
	}

	ctx := context.Background()

	err := m.SendOTP(ctx, "calon@example.com", "Budi Santoso", "123456")
	if err != nil {
		t.Errorf("SendOTP pada logMailer tidak boleh error: %v", err)
	}

	err = m.SendPasswordReset(ctx, "calon@example.com", "Budi Santoso", "http://localhost:5173/reset-password?token=abc")
	if err != nil {
		t.Errorf("SendPasswordReset pada logMailer tidak boleh error: %v", err)
	}
}

func TestNewMailer(t *testing.T) {
	log := zap.NewNop()

	// Driver log
	cfgLog := &config.Config{
		MailDriver: "log",
		AppName:    "PMB Universitas",
	}
	mailerLog := NewMailer(cfgLog, log)
	if mailerLog == nil {
		t.Fatal("NewMailer dengan driver log tidak boleh nil")
	}

	// Driver smtp (Mailpit)
	cfgSMTP := &config.Config{
		MailDriver:      "smtp",
		MailHost:        "localhost",
		MailPort:        1025,
		MailFromAddress: "pmb@kampus.ac.id",
		MailFromName:    "PMB Universitas",
		AppName:         "PMB Universitas",
	}
	mailerSMTP := NewMailer(cfgSMTP, log)
	if mailerSMTP == nil {
		t.Fatal("NewMailer dengan driver smtp tidak boleh nil")
	}
}
