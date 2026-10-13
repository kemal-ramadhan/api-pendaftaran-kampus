package pdf

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWithdrawalLetterPDF(t *testing.T) {
	data := WithdrawalLetterData{
		LetterNumber:       "PMB/UND/2026/0001",
		FullName:           "Budi Santoso",
		RegistrationNumber: "REG-2026-0001",
		ProgramName:        "Teknik Informatika",
		PathName:           "Jalur Reguler",
		StageAtWithdrawal:  "verified",
		ReasonCategory:     "Diterima di Perguruan Tinggi Lain",
		ReasonText:         "Saya telah diterima di universitas negeri melalui jalur beasiswa penuh sehingga mengundurkan diri.",
		EffectiveDate:      time.Now(),
		VerifyURL:          "https://pmb.kampus.ac.id/admission/withdrawals/verify/abcd1234efgh5678",
	}

	pdfBytes, err := GenerateWithdrawalLetterPDF(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	// Validasi header PDF
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF-1.4"))
}
