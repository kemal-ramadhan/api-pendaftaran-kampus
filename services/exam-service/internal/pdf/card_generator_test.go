package pdf

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateExamCardPDF(t *testing.T) {
	data := ExamCardData{
		ParticipantNumber: "EXAM-2026-00001",
		FullName:          "Budi Santoso",
		ApplicationNumber: "PMB-2026-REG-00001",
		PeriodName:        "Gelombang 1 2026",
		PathName:          "Jalur Reguler",
		StudyProgramName:  "Teknik Informatika",
		SessionTitle:      "Ujian Tulis Kemampuan Dasar",
		SessionType:       "written_test",
		StartAt:           time.Now().Add(24 * time.Hour),
		EndAt:             time.Now().Add(26 * time.Hour),
		Location:          "Gedung Rektorat Lt. 3 R.301",
		MeetingLink:       "",
	}

	pdfBytes, err := GenerateExamCardPDF(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Verifikasi header %PDF-1.4 dan footer %%EOF
	content := string(pdfBytes)
	assert.True(t, strings.HasPrefix(content, "%PDF-1.4"))
	assert.True(t, strings.Contains(content, "%%EOF"))
	assert.True(t, strings.Contains(content, "EXAM-2026-00001"))
	assert.True(t, strings.Contains(content, "Budi Santoso"))
}
