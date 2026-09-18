package utils

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) error {
	host := os.Getenv("MAIL_HOST")
	portStr := os.Getenv("MAIL_PORT")
	username := os.Getenv("MAIL_USERNAME")
	password := os.Getenv("MAIL_PASSWORD")
	from := os.Getenv("MAIL_FROM_ADDRESS")

	port, _ := strconv.Atoi(portStr)

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(host, port, username, password)

	if err := d.DialAndSend(m); err != nil {
		log.Println("Error sending email:", err)
		return err
	}

	return nil
}

func SendOtpEmail(to string, otp int) error {
	// Ambil variabel dari .env
	host := os.Getenv("MAIL_HOST")
	portStr := os.Getenv("MAIL_PORT")
	username := os.Getenv("MAIL_USERNAME")
	password := os.Getenv("MAIL_PASSWORD")
	from := os.Getenv("MAIL_FROM_ADDRESS")
	fromName := os.Getenv("MAIL_FROM_NAME")

	// Validasi variabel env
	if host == "" || portStr == "" || username == "" || password == "" || from == "" {
		return fmt.Errorf("email environment variables not set properly")
	}

	// Konversi port ke int
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 587
	}

	// Format body email
	subject := "Your OTP Code - Jatidiri App"
	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>OTP Jatidiri App</title>
	</head>
	<body style="font-family: Arial, sans-serif; background-color: #f4f6f8; padding: 0; margin: 0;">
		<div style="max-width: 600px; margin: 40px auto; background-color: #ffffff; border-radius: 10px; overflow: hidden; box-shadow: 0 4px 10px rgba(0,0,0,0.1);">
			
			<!-- Header -->
			<div style="background-color: #2e86de; color: #fff; text-align: center; padding: 20px 0;">
				<h1 style="margin: 0; font-size: 24px;">Jatidiri App</h1>
			</div>

			<!-- Body -->
			<div style="padding: 30px;">
				<h2 style="color: #333;">Hai, %s 👋</h2>
				<p style="color: #555; font-size: 16px;">Terima kasih telah mendaftar di <b>Jatidiri App</b>.</p>
				<p style="color: #555; font-size: 16px;">Berikut kode OTP kamu:</p>

				<div style="background-color: #f0f4ff; border: 2px dashed #2e86de; border-radius: 8px; padding: 15px; text-align: center; margin: 20px 0;">
					<span style="font-size: 28px; font-weight: bold; color: #2e86de; letter-spacing: 4px;">%06d</span>
				</div>

				<p style="color: #666; font-size: 14px;">Kode ini hanya berlaku selama <b>5 menit</b>. Jangan bagikan kode ini kepada siapa pun untuk menjaga keamanan akunmu.</p>

				<p style="color: #555; font-size: 15px; margin-top: 25px;">Salam hangat,<br><b>Tim Jatidiri App</b></p>
			</div>

			<!-- Footer -->
			<div style="background-color: #f4f6f8; color: #888; text-align: center; font-size: 12px; padding: 15px;">
				<p style="margin: 0;">&copy; 2025 Jatidiri App. All rights reserved.</p>
			</div>

		</div>
	</body>
	</html>
	`, to, otp)

	// Setup email
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", fromName, from))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// Kirim email
	d := gomail.NewDialer(host, port, username, password)
	d.SSL = false // pakai STARTTLS (port 587)

	if err := d.DialAndSend(m); err != nil {
		log.Println("❌ Failed to send email:", err)
		return err
	}

	log.Println("✅ OTP email sent to:", to)
	return nil
}

func SendResetPasswordEmail(email string, link string) error {

	subject := "Reset Password"

	body := fmt.Sprintf(`
Halo,

Klik link berikut untuk reset password:

%s

Link ini berlaku selama 30 menit.

Jika bukan Anda yang meminta reset password, abaikan email ini.
`, link)

	return SendEmail(email, subject, body)
}
