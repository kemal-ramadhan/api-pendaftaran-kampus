package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

type WithdrawalLetterData struct {
	LetterNumber       string
	FullName           string
	RegistrationNumber string
	ProgramName        string
	PathName           string
	StageAtWithdrawal  string
	ReasonCategory     string
	ReasonText         string
	EffectiveDate      time.Time
	VerifyURL          string
}

// GenerateWithdrawalLetterPDF menghasilkan dokumen PDF valid surat bukti pengunduran diri
func GenerateWithdrawalLetterPDF(data WithdrawalLetterData) ([]byte, error) {
	effectiveStr := data.EffectiveDate.Format("02 January 2006")

	var stream bytes.Buffer

	// Border halaman
	stream.WriteString("0.8 0.8 0.8 RG\n1 w\n30 30 535 782 re S\n")
	// Header banner
	stream.WriteString("0.55 0.1 0.1 rg\n30 742 535 70 re f\n")

	// Header Text (Putih)
	stream.WriteString("BT\n/F1 16 Tf\n1 1 1 rg\n50 782 Td\n(SURAT KETERANGAN PENGUNDURAN DIRI) Tj\n")
	stream.WriteString("/F1 10 Tf\n0 -18 Td\n(Penerimaan Mahasiswa Baru - Dokumen Resmi Pembatalan Status) Tj\nET\n")

	// Nomor Surat
	stream.WriteString("BT\n0 0 0 rg\n")
	stream.WriteString("/F1 11 Tf\n50 710 Td\n" + fmt.Sprintf("(NOMOR: %s) Tj\n", escapePDF(data.LetterNumber)))

	// Informasi Pendaftar
	stream.WriteString("/F1 10 Tf\n")
	stream.WriteString(fmt.Sprintf("0 -28 Td\n(Nama Lengkap               : %s) Tj\n", escapePDF(data.FullName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Nomor Pendaftaran          : %s) Tj\n", escapePDF(data.RegistrationNumber)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Program Studi              : %s) Tj\n", escapePDF(data.ProgramName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Jalur Pendaftaran          : %s) Tj\n", escapePDF(data.PathName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Tahap Saat Mundur          : %s) Tj\n", escapePDF(data.StageAtWithdrawal)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Kategori Alasan            : %s) Tj\n", escapePDF(data.ReasonCategory)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Tanggal Berlaku            : %s) Tj\n", escapePDF(effectiveStr)))

	// Alasan
	stream.WriteString("0 -26 Td\n(Pernyataan Alasan:) Tj\n")
	lines := splitReasonText(data.ReasonText, 65)
	for _, l := range lines {
		stream.WriteString(fmt.Sprintf("0 -16 Td\n(  %s) Tj\n", escapePDF(l)))
	}

	// Pernyataan Efektif
	stream.WriteString("0 -30 Td\n(Dengan terbitnya surat ini, pendaftaran yang bersangkutan dinyatakan WITHDRAWN) Tj\n")
	stream.WriteString("0 -16 Td\n(dan seluruh hak sebagai calon mahasiswa pada periode ini telah dibatalkan.) Tj\n")

	// Tanda tangan & stempel
	stream.WriteString("0 -40 Td\n(Ditetapkan oleh Panitia Penerimaan Mahasiswa Baru) Tj\n")
	stream.WriteString(fmt.Sprintf("0 -16 Td\n(Pada tanggal: %s) Tj\n", escapePDF(effectiveStr)))
	stream.WriteString("0 -16 Td\n(Dokumen ini diterbitkan secara elektronik dan sah tanpa tanda tangan basah.) Tj\n")
	stream.WriteString("ET\n")

	// Render QR Code untuk URL Verifikasi
	var qrX, qrY, qrW int
	if data.VerifyURL != "" {
		qrPng, err := qrcode.New(data.VerifyURL, qrcode.Medium)
		if err == nil {
			bm := qrPng.Bitmap()
			size := len(bm)
			qrX = 400
			qrY = 90
			qrW = 120

			scale := float64(qrW) / float64(size)
			stream.WriteString("0 0 0 rg\n")
			for y := 0; y < size; y++ {
				for x := 0; x < size; x++ {
					if bm[y][x] {
						px := float64(qrX) + float64(x)*scale
						py := float64(qrY) + float64(size-1-y)*scale
						stream.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n", px, py, scale, scale))
					}
				}
			}

			// Label QR
			stream.WriteString("BT\n0 0 0 rg\n/F1 8 Tf\n")
			stream.WriteString(fmt.Sprintf("%d %d Td\n(Pindai untuk Verifikasi Keabsahan) Tj\nET\n", qrX, qrY-14))
		}
	}

	streamBytes := stream.Bytes()

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)

	// Object 1: Catalog
	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Object 3: Page
	offsets[3] = buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// Object 4: Content Stream
	offsets[4] = buf.Len()
	buf.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n", len(streamBytes)))
	buf.Write(streamBytes)
	buf.WriteString("\nendstream\nendobj\n")

	// Object 5: Font Helvetica
	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// XRef Table
	startxref := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}

	// Trailer
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", startxref))

	return buf.Bytes(), nil
}

func escapePDF(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func splitReasonText(text string, maxLen int) []string {
	if text == "" {
		return []string{"-"}
	}
	words := strings.Fields(text)
	var lines []string
	var current strings.Builder
	for _, w := range words {
		if current.Len()+len(w)+1 > maxLen {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(w)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}
