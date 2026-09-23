package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

type ExamCardData struct {
	ParticipantNumber string
	FullName          string
	ApplicationNumber string
	PeriodName        string
	PathName          string
	StudyProgramName  string
	SessionTitle      string
	SessionType       string
	StartAt           time.Time
	EndAt             time.Time
	Location          string
	MeetingLink       string
}

// GenerateExamCardPDF menghasilkan dokumen PDF valid kartu ujian PMB
func GenerateExamCardPDF(data ExamCardData) ([]byte, error) {
	timeRange := fmt.Sprintf("%s, %s - %s WIB",
		data.StartAt.Format("02 January 2006"),
		data.StartAt.Format("15:04"),
		data.EndAt.Format("15:04"),
	)

	loc := data.Location
	if loc == "" {
		if data.MeetingLink != "" {
			loc = "Online: " + data.MeetingLink
		} else {
			loc = "Ruang Ujian Terpadu PMB"
		}
	}

	var stream bytes.Buffer

	// Border kartu
	stream.WriteString("0.8 0.8 0.8 RG\n1 w\n30 30 535 782 re S\n")
	// Header banner
	stream.WriteString("0.1 0.2 0.45 rg\n30 752 535 60 re f\n")

	// Header Text (Putih)
	stream.WriteString("BT\n/F1 16 Tf\n1 1 1 rg\n50 785 Td\n(KARTU TANDA PESERTA UJIAN PMB) Tj\n")
	stream.WriteString("/F1 10 Tf\n0 -18 Td\n(Penerimaan Mahasiswa Baru Tahun Akademik Resmi) Tj\nET\n")

	// Text Info Peserta (Hitam)
	stream.WriteString("BT\n0 0 0 rg\n")
	stream.WriteString("/F1 14 Tf\n50 710 Td\n" + fmt.Sprintf("(NOMOR PESERTA: %s) Tj\n", escapePDF(data.ParticipantNumber)))

	stream.WriteString("/F1 10 Tf\n")
	stream.WriteString(fmt.Sprintf("0 -26 Td\n(Nama Lengkap        : %s) Tj\n", escapePDF(data.FullName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Nomor Pendaftaran   : %s) Tj\n", escapePDF(data.ApplicationNumber)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Jalur Masuk         : %s) Tj\n", escapePDF(data.PathName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Gelombang           : %s) Tj\n", escapePDF(data.PeriodName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Pilihan Program Studi: %s) Tj\n", escapePDF(data.StudyProgramName)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Jenis Sesi          : %s) Tj\n", escapePDF(data.SessionType)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Judul Sesi          : %s) Tj\n", escapePDF(data.SessionTitle)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Jadwal Pelaksanaan  : %s) Tj\n", escapePDF(timeRange)))
	stream.WriteString(fmt.Sprintf("0 -20 Td\n(Lokasi / Tautan     : %s) Tj\n", escapePDF(loc)))
	stream.WriteString("ET\n")

	// Pasfoto box di kanan atas
	stream.WriteString("0.7 0.7 0.7 RG\n420 570 120 150 re S\n")
	stream.WriteString("BT\n/F1 9 Tf\n0.5 0.5 0.5 rg\n440 645 Td\n(FOTO PESERTA) Tj\n0 -15 Td\n(RESMI 3x4) Tj\nET\n")

	// Vector QR Code di kanan bawah foto
	qrSize := 120.0
	qrX := 420.0
	qrY := 420.0

	qr, err := qrcode.New(data.ParticipantNumber, qrcode.Medium)
	if err == nil {
		bitmap := qr.Bitmap()
		numRows := len(bitmap)
		if numRows > 0 {
			moduleSize := qrSize / float64(numRows)
			stream.WriteString("0 0 0 rg\n")
			for y := 0; y < numRows; y++ {
				for x := 0; x < len(bitmap[y]); x++ {
					if bitmap[y][x] {
						// PDF coordinates: origin bottom-left
						posX := qrX + float64(x)*moduleSize
						posY := qrY + float64(numRows-1-y)*moduleSize
						stream.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n", posX, posY, moduleSize, moduleSize))
					}
				}
			}
		}
	}

	// Tata tertib ujian
	stream.WriteString("0.9 0.9 0.95 rg\n50 160 495 180 re f\n")
	stream.WriteString("0.8 0.8 0.9 RG\n50 160 495 180 re S\n")

	stream.WriteString("BT\n0.1 0.1 0.1 rg\n/F1 11 Tf\n65 315 Td\n(TATA TERTIB PESERTA UJIAN:) Tj\n")
	stream.WriteString("/F1 9 Tf\n")
	stream.WriteString("0 -20 Td\n(1. Peserta wajib mencetak dan membawa Kartu Tanda Peserta ini saat ujian berlangsung.) Tj\n")
	stream.WriteString("0 -16 Td\n(2. Peserta wajib membawa identitas diri resmi berfoto (KTP / SIM / Kartu Pelajar).) Tj\n")
	stream.WriteString("0 -16 Td\n(3. Peserta hadir di lokasi ujian atau bergabung di tautan virtual paling lambat 15 menit sebelum ujian.) Tj\n")
	stream.WriteString("0 -16 Td\n(4. Tunjukkan kode QR pada kartu kepada pengawas ujian untuk proses verifikasi presensi.) Tj\n")
	stream.WriteString("0 -16 Td\n(5. Kecurangan dalam bentuk apa pun akan mengakibatkan diskualifikasi seketika.) Tj\n")
	stream.WriteString("ET\n")

	// Footer catatan elektronik
	stream.WriteString("BT\n/F1 8 Tf\n0.4 0.4 0.4 rg\n50 60 Td\n(Dokumen ini diterbitkan sah secara elektronik oleh Sistem Seleksi PMB. Nomor Peserta terverifikasi digital.) Tj\nET\n")

	content := stream.String()

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	// Objek 1: Catalog
	obj1Offset := buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Objek 2: Pages
	obj2Offset := buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Objek 3: Page (A4 595 x 842 pt)
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
