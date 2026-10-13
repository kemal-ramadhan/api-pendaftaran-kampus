package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type Mailer interface {
	SendOTP(ctx context.Context, toEmail, toName, otpCode string) error
	SendPasswordReset(ctx context.Context, toEmail, toName, resetLink string) error
}

type logMailer struct {
	appName string
	log     *zap.Logger
}

func (m *logMailer) SendOTP(ctx context.Context, toEmail, toName, otpCode string) error {
	m.log.Info("📧 [MAIL_DRIVER=log] Mengirim OTP Email",
		zap.String("to_email", toEmail),
		zap.String("to_name", toName),
		zap.String("otp_code", otpCode),
		zap.String("app_name", m.appName),
	)
	return nil
}

func (m *logMailer) SendPasswordReset(ctx context.Context, toEmail, toName, resetLink string) error {
	m.log.Info("📧 [MAIL_DRIVER=log] Mengirim Link Reset Password",
		zap.String("to_email", toEmail),
		zap.String("to_name", toName),
		zap.String("reset_link", resetLink),
		zap.String("app_name", m.appName),
	)
	return nil
}

type smtpMailer struct {
	cfg *config.Config
	log *zap.Logger
}

func (m *smtpMailer) SendOTP(ctx context.Context, toEmail, toName, otpCode string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", fmt.Sprintf("%s <%s>", m.cfg.MailFromName, m.cfg.MailFromAddress))
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", fmt.Sprintf("Kode Verifikasi Pendaftaran - %s", m.cfg.AppName))

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head><meta charset="utf-8"></head>
	<body style="font-family: sans-serif; line-height: 1.6; color: #333;">
		<h2>Halo, %s</h2>
		<p>Terima kasih telah mendaftar di <strong>%s</strong>.</p>
		<p>Berikut adalah kode OTP verifikasi akun Anda:</p>
		<div style="background: #f4f4f4; padding: 15px; border-radius: 8px; font-size: 24px; font-weight: bold; letter-spacing: 5px; text-align: center; margin: 20px 0;">
			%s
		</div>
		<p>Kode ini berlaku selama 10 menit. Jangan bagikan kode ini kepada siapa pun.</p>
		<br/>
		<p>Salam hangat,<br/>Tim PMB %s</p>
	</body>
	</html>
	`, toName, m.cfg.AppName, otpCode, m.cfg.AppName)

	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(m.cfg.MailHost, m.cfg.MailPort, m.cfg.MailUsername, m.cfg.MailPassword)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if m.cfg.MailUsername == "" && m.cfg.MailPassword == "" {
		dialer.Auth = smtp.Auth(nil)
	}

	return dialer.DialAndSend(msg)
}

func (m *smtpMailer) SendPasswordReset(ctx context.Context, toEmail, toName, resetLink string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", fmt.Sprintf("%s <%s>", m.cfg.MailFromName, m.cfg.MailFromAddress))
	msg.SetHeader("To", toEmail)
	msg.SetHeader("Subject", fmt.Sprintf("Atur Ulang Kata Sandi - %s", m.cfg.AppName))

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head><meta charset="utf-8"></head>
	<body style="font-family: sans-serif; line-height: 1.6; color: #333;">
		<h2>Halo, %s</h2>
		<p>Kami menerima permintaan untuk mengatur ulang kata sandi akun Anda di <strong>%s</strong>.</p>
		<p>Klik tombol di bawah ini untuk membuat kata sandi baru:</p>
		<div style="text-align: center; margin: 25px 0;">
			<a href="%s" style="background: #2563eb; color: #fff; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: bold; display: inline-block;">
				Reset Kata Sandi
			</a>
		</div>
		<p>Atau salin tautan berikut ke peramban Anda:</p>
		<p><a href="%s">%s</a></p>
		<p>Tautan ini berlaku selama 30 menit. Jika Anda tidak merasa meminta reset kata sandi, abaikan email ini.</p>
		<br/>
		<p>Salam hangat,<br/>Tim PMB %s</p>
	</body>
	</html>
	`, toName, m.cfg.AppName, resetLink, resetLink, resetLink, m.cfg.AppName)

	msg.SetBody("text/html", body)

	dialer := gomail.NewDialer(m.cfg.MailHost, m.cfg.MailPort, m.cfg.MailUsername, m.cfg.MailPassword)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if m.cfg.MailUsername == "" && m.cfg.MailPassword == "" {
		dialer.Auth = smtp.Auth(nil)
	}

	return dialer.DialAndSend(msg)
}

func NewMailer(cfg *config.Config, log *zap.Logger) Mailer {
	if strings.ToLower(cfg.MailDriver) == "smtp" {
		log.Info("Menggunakan SMTP Mailer", zap.String("host", cfg.MailHost), zap.Int("port", cfg.MailPort))
		return &smtpMailer{cfg: cfg, log: log}
	}
	log.Info("Menggunakan Log Mailer (development)")
	return &logMailer{appName: cfg.AppName, log: log}
}
