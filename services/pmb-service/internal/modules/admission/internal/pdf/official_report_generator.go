package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type InstitutionHeaderData struct {
	FoundationName      string
	InstitutionName     string
	DocumentCode        string
	Address             string
	City                string
	Phone               string
	Email               string
	CommitteeChairName  string
	CommitteeChairTitle string
	CommitteeChairNIP   string
}

type OfficialReportItem struct {
	Col1 string // No Reg / Inv
	Col2 string // Nama
	Col3 string // Prodi / Kanal
	Col4 string // Jalur / Jenis
	Col5 string // Status / Nominal
}

type OfficialReportData struct {
	Institution InstitutionHeaderData
	DocType     string // "registrant", "selection", "reconciliation"
	Title       string
	DocNumber   string
	PeriodName  string
	PrintedAt   time.Time
	TotalCount  int
	SummaryText string
	Items       []OfficialReportItem
}

func escapePDFText(text string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"(", "\\(",
		")", "\\)",
		"\r", "",
		"\n", " ",
	)
	return r.Replace(text)
}

// GenerateOfficialReportPDF menghasilkan dokumen PDF resmi berita acara
func GenerateOfficialReportPDF(data OfficialReportData) ([]byte, error) {
	city := data.Institution.City
	if city == "" {
		city = "Tasikmalaya"
	}
	chairTitle := data.Institution.CommitteeChairTitle
	if chairTitle == "" {
		chairTitle = fmt.Sprintf("Ketua Panitia PMB %d", data.PrintedAt.Year())
	}
	chairName := data.Institution.CommitteeChairName
	if chairName == "" {
		chairName = "Dr. H. Ahmad Fauzi, M.Kes."
	}
	chairNIP := data.Institution.CommitteeChairNIP
	if chairNIP == "" {
		chairNIP = "19750812 200212 1 002"
	}
	foundation := data.Institution.FoundationName
	if foundation == "" {
		foundation = "YAYASAN BAKTI TUNAS HUSADA"
	}
	univ := data.Institution.InstitutionName
	if univ == "" {
		univ = "UNIVERSITAS BAKTI TUNAS HUSADA"
	}

	dateStr := data.PrintedAt.Format("02 January 2006")
	timeStr := data.PrintedAt.Format("15:04 WIB")

	var stream bytes.Buffer

	// Page Border
	stream.WriteString("0.8 0.8 0.8 RG\n0.8 w\n30 30 535 782 re S\n")

	// Header Kop Institusi (Top Banner)
	stream.WriteString("0.08 0.22 0.38 rg\n30 740 535 72 re f\n")

	// Header Text Putih
	stream.WriteString("BT\n/F1 9 Tf\n1 1 1 rg\n50 788 Td\n" + fmt.Sprintf("(%s) Tj\n", escapePDFText(strings.ToUpper(foundation))))
	stream.WriteString("/F1 15 Tf\n0 -18 Td\n" + fmt.Sprintf("(%s) Tj\n", escapePDFText(strings.ToUpper(univ))))
	stream.WriteString("/F1 8 Tf\n0 -14 Td\n" + fmt.Sprintf("(Pusat Admisi & Penerimaan Mahasiswa Baru • %s • Telp: %s) Tj\n", escapePDFText(data.Institution.Address), escapePDFText(data.Institution.Phone)))
	stream.WriteString("ET\n")

	// Judul Berita Acara & Nomor
	stream.WriteString("BT\n0 0 0 rg\n")
	stream.WriteString("/F1 13 Tf\n50 710 Td\n" + fmt.Sprintf("(%s) Tj\n", escapePDFText(strings.ToUpper(data.Title))))
	stream.WriteString("/F1 10 Tf\n0 -16 Td\n" + fmt.Sprintf("(NOMOR: %s) Tj\n", escapePDFText(data.DocNumber)))
	stream.WriteString("/F1 8.5 Tf\n0 -14 Td\n" + fmt.Sprintf("(Periode: %s | Total Catatan: %d Data) Tj\n", escapePDFText(data.PeriodName), data.TotalCount))
	stream.WriteString("ET\n")

	// Garis Pemisah
	stream.WriteString("0.2 0.2 0.2 RG\n1 w\n50 660 m 515 660 l S\n")

	// Keterangan / Summary Text
	stream.WriteString("BT\n0.2 0.2 0.2 rg\n/F1 8.5 Tf\n50 644 Td\n")
	summaryLines := escapePDFText(data.SummaryText)
	if summaryLines == "" {
		summaryLines = "Dokumen resmi ini merupakan rekapan sah hasil proses verifikasi administratif Sistem PMB."
	}
	stream.WriteString(fmt.Sprintf("(%s) Tj\nET\n", summaryLines))

	// Header Tabel Ringkas
	stream.WriteString("0.92 0.94 0.96 rg\n50 605 465 18 re f\n")
	stream.WriteString("0.7 0.7 0.7 RG\n0.5 w\n50 605 465 18 re S\n")

	stream.WriteString("BT\n0 0 0 rg\n/F1 8 Tf\n55 611 Td\n(NO/KODE) Tj\n")
	stream.WriteString("95 0 Td\n(NAMA LENGKAP / DESKRIPSI) Tj\n")
	stream.WriteString("160 0 Td\n(PRODI / KANAL) Tj\n")
	stream.WriteString("100 0 Td\n(JALUR / JENIS) Tj\n")
	stream.WriteString("60 0 Td\n(STATUS / TOTAL) Tj\n")
	stream.WriteString("ET\n")

	// Render hingga 12 item
	curY := 590
	countRender := len(data.Items)
	if countRender > 12 {
		countRender = 12
	}
	for i := 0; i < countRender; i++ {
		item := data.Items[i]
		if i%2 == 1 {
			stream.WriteString(fmt.Sprintf("0.97 0.97 0.98 rg\n50 %d 465 15 re f\n", curY-3))
		}
		stream.WriteString(fmt.Sprintf("BT\n0.15 0.15 0.15 rg\n/F1 7.5 Tf\n55 %d Td\n(%s) Tj\n", curY, escapePDFText(item.Col1)))
		stream.WriteString(fmt.Sprintf("95 0 Td\n(%s) Tj\n", escapePDFText(item.Col2)))
		stream.WriteString(fmt.Sprintf("160 0 Td\n(%s) Tj\n", escapePDFText(item.Col3)))
		stream.WriteString(fmt.Sprintf("100 0 Td\n(%s) Tj\n", escapePDFText(item.Col4)))
		stream.WriteString(fmt.Sprintf("60 0 Td\n(%s) Tj\n", escapePDFText(item.Col5)))
		stream.WriteString("ET\n")
		curY -= 16
	}

	if len(data.Items) > 12 {
		stream.WriteString(fmt.Sprintf("BT\n0.4 0.4 0.4 rg\n/F1 7.5 Tf\n55 %d Td\n(... dan %d rekaman data lainnya terlampir pada basis data sistem PMB) Tj\nET\n", curY, len(data.Items)-12))
	}

	// Tanda Tangan & Pengesahan di Bawah
	sigY := 190
	stream.WriteString("BT\n0 0 0 rg\n/F1 9 Tf\n")
	stream.WriteString(fmt.Sprintf("340 %d Td\n(%s, %s) Tj\n", sigY, escapePDFText(city), escapePDFText(dateStr)))
	stream.WriteString(fmt.Sprintf("0 -15 Td\n(%s) Tj\n", escapePDFText(chairTitle)))
	stream.WriteString("0 -50 Td\n")
	stream.WriteString(fmt.Sprintf("(%s) Tj\n", escapePDFText(chairName)))
	stream.WriteString("/F1 8 Tf\n")
	stream.WriteString(fmt.Sprintf("0 -12 Td\n(NIP: %s) Tj\n", escapePDFText(chairNIP)))
	stream.WriteString("ET\n")

	// Garis Bawah Nama TTD
	stream.WriteString(fmt.Sprintf("0 0 0 RG\n0.5 w\n340 %d m 485 %d l S\n", sigY-68, sigY-68))

	// Footer Watermark & Cap Waktu
	stream.WriteString("BT\n0.5 0.5 0.5 rg\n/F1 7.5 Tf\n50 45 Td\n")
	stream.WriteString(fmt.Sprintf("(Dicetak melalui Sistem PMB: %s, %s | Dokumen Sah Berita Acara PMB) Tj\nET\n", escapePDFText(dateStr), escapePDFText(timeStr)))

	// Bungkus PDF
	content := stream.Bytes()
	contentLen := len(content)

	var doc bytes.Buffer
	doc.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)

	// Object 1: Catalog
	offsets[1] = doc.Len()
	doc.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets[2] = doc.Len()
	doc.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Object 3: Page
	offsets[3] = doc.Len()
	doc.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// Object 4: Stream Content
	offsets[4] = doc.Len()
	doc.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n", contentLen))
	doc.Write(content)
	doc.WriteString("\nendstream\nendobj\n")

	// Object 5: Standard Helvetica Font
	offsets[5] = doc.Len()
	doc.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// XRef Table
	xrefOffset := doc.Len()
	doc.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		doc.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}

	// Trailer
	doc.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))

	return doc.Bytes(), nil
}
