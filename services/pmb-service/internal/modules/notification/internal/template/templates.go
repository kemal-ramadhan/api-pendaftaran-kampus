package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
)

type RenderedNotification struct {
	Subject string
	Body    string
}

type TemplateEngine interface {
	RenderEmail(templateName string, rawData string) (*RenderedNotification, error)
	RenderWhatsApp(templateName string, rawData string) (string, error)
}

type templateEngine struct{}

func NewTemplateEngine() TemplateEngine {
	return &templateEngine{}
}

func parseData(rawData string) map[string]interface{} {
	data := make(map[string]interface{})
	if rawData != "" {
		_ = json.Unmarshal([]byte(rawData), &data)
	}
	return data
}

func (t *templateEngine) RenderEmail(templateName string, rawData string) (*RenderedNotification, error) {
	data := parseData(rawData)

	switch templateName {
	case "email_verification":
		subject := "Verifikasi Email Pendaftaran PMB"
		tmplStr := `Halo {{.name}},<br><br>Terima kasih telah mendaftar di sistem Penerimaan Mahasiswa Baru (PMB).<br>Silakan klik tautan berikut untuk memverifikasi akun Anda:<br><a href="{{.verification_url}}">{{.verification_url}}</a><br><br>Kode verifikasi: <b>{{.code}}</b><br><br>Salam hangat,<br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	case "application_registered":
		subject := fmt.Sprintf("Pendaftaran Berhasil - %v", data["registration_no"])
		tmplStr := `Halo {{.name}},<br><br>Pendaftaran Anda dengan nomor registrasi <b>{{.registration_no}}</b> telah berhasil dibuat pada jalur <b>{{.path_name}}</b> ({{.period_name}}).<br><br>Silakan lakukan pembayaran biaya pendaftaran sebesar <b>Rp {{.bill_amount}}</b> melalui VA <b>{{.va_number}}</b> sebelum batas waktu <b>{{.payment_deadline}}</b>.<br><br>Terima kasih,<br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	case "payment_received":
		subject := fmt.Sprintf("Pembayaran Diterima - %v", data["registration_no"])
		tmplStr := `Halo {{.name}},<br><br>Pembayaran biaya pendaftaran untuk nomor registrasi <b>{{.registration_no}}</b> sejumlah <b>Rp {{.amount}}</b> telah berhasil kami terima pada {{.payment_date}}.<br>Silakan lengkapi berkas dan persyaratan pendaftaran di portal PMB.<br><br>Terima kasih,<br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	case "selection_passed":
		subject := "Selamat! Anda Dinyatakan Lulus Seleksi PMB"
		tmplStr := `Halo {{.name}},<br><br>Selamat! Nomor registrasi <b>{{.registration_no}}</b> dinyatakan <b>LULUS SELEKSI</b> pada program studi <b>{{.study_program}}</b>.<br>Silakan lakukan registrasi ulang melalui portal PMB sesuai jadwal.<br><br>Salam hangat,<br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	case "selection_failed":
		subject := "Pengumuman Hasil Seleksi PMB"
		tmplStr := `Halo {{.name}},<br><br>Terima kasih atas partisipasi Anda dalam seleksi PMB dengan nomor registrasi <b>{{.registration_no}}</b>.<br>Mohon maaf, saat ini Anda belum berhasil lolos pada seleksi jalur ini. Tetap semangat dan jangan putus asa.<br><br>Salam hangat,<br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	case "exam_card_ready":
		subject := fmt.Sprintf("Kartu Ujian PMB Siap Diunduh - %v", data["registration_no"])
		tmplStr := `Halo {{.name}},<br><br>Kartu ujian untuk nomor registrasi <b>{{.registration_no}}</b> telah siap diunduh di portal PMB.<br>Jadwal Ujian: <b>{{.exam_date}}</b><br>Lokasi: <b>{{.exam_location}}</b><br><br>Panitia PMB`
		body, err := renderHTML(tmplStr, data)
		return &RenderedNotification{Subject: subject, Body: body}, err

	default:
		subjectVal, ok := data["subject"].(string)
		if !ok || subjectVal == "" {
			subjectVal = "Pemberitahuan PMB"
		}
		messageVal, ok := data["message"].(string)
		if !ok {
			messageVal = "Halo, Anda memiliki pemberitahuan baru terkait PMB."
		}
		return &RenderedNotification{Subject: subjectVal, Body: messageVal}, nil
	}
}

func (t *templateEngine) RenderWhatsApp(templateName string, rawData string) (string, error) {
	data := parseData(rawData)

	switch templateName {
	case "email_verification":
		return fmt.Sprintf("Halo %v, kode verifikasi PMB Anda adalah: *%v*. Jangan berikan kode ini kepada siapapun.", data["name"], data["code"]), nil

	case "application_registered":
		return fmt.Sprintf("Halo %v, pendaftaran PMB Anda berhasil dibuat dengan No. Registrasi: *%v*. Silakan bayar tagihan Rp %v via VA %v sebelum %v.",
			data["name"], data["registration_no"], data["bill_amount"], data["va_number"], data["payment_deadline"]), nil

	case "payment_received":
		return fmt.Sprintf("Halo %v, pembayaran pendaftaran No. *%v* sebesar Rp %v telah kami terima. Silakan lengkapi berkas di portal PMB.",
			data["name"], data["registration_no"], data["amount"]), nil

	case "selection_passed":
		return fmt.Sprintf("Selamat %v! Anda dinyatakan *LULUS SELEKSI* PMB pada prodi *%v* (No. Reg: %v). Silakan cek portal untuk daftar ulang.",
			data["name"], data["study_program"], data["registration_no"]), nil

	case "selection_failed":
		return fmt.Sprintf("Halo %v, mohon maaf Anda belum dinyatakan lolos pada seleksi PMB (No. Reg: %v). Terima kasih atas partisipasi Anda.",
			data["name"], data["registration_no"]), nil

	case "exam_card_ready":
		return fmt.Sprintf("Halo %v, kartu ujian PMB No. *%v* sudah siap diunduh di portal. Jadwal: %v di %v.",
			data["name"], data["registration_no"], data["exam_date"], data["exam_location"]), nil

	default:
		if msg, ok := data["message"].(string); ok && msg != "" {
			return msg, nil
		}
		return fmt.Sprintf("Halo %v, ada pembaruan status pendaftaran PMB Anda.", data["name"]), nil
	}
}

func renderHTML(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("notification").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
