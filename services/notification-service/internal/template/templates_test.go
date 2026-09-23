package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateEngine_RenderEmail(t *testing.T) {
	engine := NewTemplateEngine()

	t.Run("render email_verification", func(t *testing.T) {
		rawData := `{"name":"Budi","code":"123456","verification_url":"https://pmb.ac.id/verify"}`
		res, err := engine.RenderEmail("email_verification", rawData)
		assert.NoError(t, err)
		assert.Equal(t, "Verifikasi Email Pendaftaran PMB", res.Subject)
		assert.True(t, strings.Contains(res.Body, "Budi"))
		assert.True(t, strings.Contains(res.Body, "123456"))
	})

	t.Run("render application_registered", func(t *testing.T) {
		rawData := `{"name":"Siti","registration_no":"PMB-2026-1-00001","path_name":"Reguler","period_name":"Gelombang 1","bill_amount":"250.000","va_number":"988123456","payment_deadline":"2026-08-01"}`
		res, err := engine.RenderEmail("application_registered", rawData)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(res.Subject, "PMB-2026-1-00001"))
		assert.True(t, strings.Contains(res.Body, "PMB-2026-1-00001"))
		assert.True(t, strings.Contains(res.Body, "988123456"))
	})

	t.Run("render default fallback template", func(t *testing.T) {
		rawData := `{"subject":"Pengumuman Khusus","message":"Ini pesan pemberitahuan"}`
		res, err := engine.RenderEmail("unknown_template", rawData)
		assert.NoError(t, err)
		assert.Equal(t, "Pengumuman Khusus", res.Subject)
		assert.Equal(t, "Ini pesan pemberitahuan", res.Body)
	})
}

func TestTemplateEngine_RenderWhatsApp(t *testing.T) {
	engine := NewTemplateEngine()

	t.Run("render application_registered whatsapp", func(t *testing.T) {
		rawData := `{"name":"Budi","registration_no":"PMB-2026-1-00001","bill_amount":"250.000","va_number":"988123456","payment_deadline":"2026-08-01"}`
		text, err := engine.RenderWhatsApp("application_registered", rawData)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(text, "Budi"))
		assert.True(t, strings.Contains(text, "PMB-2026-1-00001"))
	})

	t.Run("render selection_passed whatsapp", func(t *testing.T) {
		rawData := `{"name":"Ahmad","study_program":"Teknik Informatika","registration_no":"PMB-2026-1-00002"}`
		text, err := engine.RenderWhatsApp("selection_passed", rawData)
		assert.NoError(t, err)
		assert.True(t, strings.Contains(text, "Teknik Informatika"))
	})
}
