package pdf

import (
	"bytes"
	"fmt"
	"time"
)

type ReceiptData struct {
	ReceiptNumber string
	InvoiceNumber string
	UserName      string
	Description   string
	Amount        int64
	PaidAt        time.Time
	Method        string
}

// GenerateReceiptPDF menghasilkan dokumen PDF valid sederhana tanpa dependensi eksternal CGO
func GenerateReceiptPDF(data ReceiptData) ([]byte, error) {
	paidAtFormatted := data.PaidAt.Format("02 January 2006, 15:04 WIB")
	amountFormatted := fmt.Sprintf("Rp %s", formatRupiah(data.Amount))

	content := fmt.Sprintf(
		"BT\n/F1 18 Tf\n50 750 Td\n(KWITANSI PEMBAYARAN PMB RESMI) Tj\n"+
			"/F1 11 Tf\n0 -30 Td\n(No. Kwitansi: %s) Tj\n"+
			"0 -20 Td\n(No. Invoice : %s) Tj\n"+
			"0 -20 Td\n(Tanggal      : %s) Tj\n"+
			"0 -20 Td\n(Metode Bayar: %s) Tj\n"+
			"0 -20 Td\n(Keterangan  : %s) Tj\n"+
			"0 -30 Td\n/F1 14 Tf\n(Jumlah Bayar: %s) Tj\n"+
			"0 -40 Td\n/F1 12 Tf\n(STATUS: LUNAS) Tj\n"+
			"0 -30 Td\n/F1 10 Tf\n(Dokumen ini diterbitkan sah secara elektronik oleh Sistem PMB Kampus.) Tj\nET",
		escapePDF(data.ReceiptNumber),
		escapePDF(data.InvoiceNumber),
		escapePDF(paidAtFormatted),
		escapePDF(data.Method),
		escapePDF(data.Description),
		escapePDF(amountFormatted),
	)

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	// Objek 1: Catalog
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Objek 2: Pages
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Objek 3: Page
	obj3Offset := buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// Objek 4: Contents
	obj4Offset := buf.Len()
	buf.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content))

	// Objek 5: Font Helvetica
	obj5Offset := buf.Len()
	buf.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// xref table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n")
	buf.WriteString("0000000000 65535 f \n")
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj1Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj2Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj3Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj4Offset))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", obj5Offset))

	// Trailer
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))

	return buf.Bytes(), nil
}

func escapePDF(s string) string {
	var out bytes.Buffer
	for _, r := range s {
		switch r {
		case '(', ')', '\\':
			out.WriteRune('\\')
			out.WriteRune(r)
		default:
			if r >= 32 && r <= 126 {
				out.WriteRune(r)
			} else {
				out.WriteRune(' ')
			}
		}
	}
	return out.String()
}

func formatRupiah(amount int64) string {
	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	var res []byte
	rem := n % 3
	if rem > 0 {
		res = append(res, str[:rem]...)
		if rem < n {
			res = append(res, '.')
		}
	}

	for i := rem; i < n; i += 3 {
		res = append(res, str[i:i+3]...)
		if i+3 < n {
			res = append(res, '.')
		}
	}
	return string(res)
}
