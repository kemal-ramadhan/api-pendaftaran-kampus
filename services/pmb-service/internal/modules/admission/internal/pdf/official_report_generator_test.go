package pdf

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateOfficialReportPDF(t *testing.T) {
	data := OfficialReportData{
		Institution: InstitutionHeaderData{
			FoundationName:      "Yayasan Bakti Tunas Husada",
			InstitutionName:     "Universitas Bakti Tunas Husada",
			DocumentCode:        "UBTH",
			Address:             "Jl. Cilolohan No. 36",
			City:                "Tasikmalaya",
			Phone:               "0265-337769",
			CommitteeChairName:  "Dr. H. Ahmad Fauzi",
			CommitteeChairTitle: "Ketua Panitia PMB 2026",
			CommitteeChairNIP:   "19750812 200212 1 002",
		},
		DocType:     "registrant",
		Title:       "Berita Acara Laporan Pendaftar PMB",
		DocNumber:   "001/BA-PMB/UBTH/IX/2026",
		PeriodName:  "Gelombang 1 PMB 2026/2027",
		PrintedAt:   time.Now(),
		TotalCount:  1,
		SummaryText: "Total 1 pendaftar tercatat dalam sistem admisi.",
		Items: []OfficialReportItem{
			{
				Col1: "REG-2026-0001",
				Col2: "Ahmad Dahlan",
				Col3: "S1 Farmasi",
				Col4: "Reguler",
				Col5: "Lulus Seleksi",
			},
		},
	}

	pdfBytes, err := GenerateOfficialReportPDF(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)

	str := string(pdfBytes)
	assert.True(t, strings.HasPrefix(str, "%PDF-1.4"))
	assert.True(t, strings.Contains(str, "001/BA-PMB/UBTH/IX/2026"))
	assert.True(t, strings.Contains(str, "%%EOF"))
}
