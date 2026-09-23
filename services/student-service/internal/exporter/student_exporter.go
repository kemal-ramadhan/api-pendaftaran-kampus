package exporter

import (
	"bytes"
	"fmt"

	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"github.com/xuri/excelize/v2"
)

type StudentExporter interface {
	ExportStudents(students []model.Student) ([]byte, error)
}

type studentExporter struct{}

func NewStudentExporter() StudentExporter {
	return &studentExporter{}
}

func (e *studentExporter) ExportStudents(students []model.Student) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Mahasiswa Baru"
	f.SetSheetName("Sheet1", sheetName)

	// Style Judul
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "1F4E79"},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	// Style Header Tabel (Navy Blue)
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Style Data Standar
	dataStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Style Data Tengah (Nomor, Tahun, Status, NIM)
	centerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	// Tulis Judul
	f.SetCellValue(sheetName, "A1", "LAPORAN DATA MAHASISWA BARU HASIL DAFTAR ULANG")
	f.SetCellStyle(sheetName, "A1", "A1", titleStyle)
	f.SetRowHeight(sheetName, 1, 28)

	f.SetCellValue(sheetName, "A2", fmt.Sprintf("Total Mahasiswa Terdaftar: %d", len(students)))
	f.SetRowHeight(sheetName, 2, 20)

	// Tulis Header Tabel di baris 4
	headers := []string{"No", "NIM", "Program Studi", "Tahun Masuk", "Jalur Masuk", "Status", "User ID", "Application ID", "Tanggal Daftar Ulang"}
	f.SetRowHeight(sheetName, 4, 25)
	for i, h := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s4", colName)
		f.SetCellValue(sheetName, cell, h)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Tulis Data Mahasiswa
	row := 5
	for idx, s := range students {
		nim := s.NIM
		if nim == "" {
			nim = "(Menunggu NIM)"
		}

		statusLabel := "Daftar Ulang"
		if s.Status == model.StudentStatusActive {
			statusLabel = "Aktif"
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), idx+1)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), nim)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), s.StudyProgramCode)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), s.EntryYear)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), s.EntryPath)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), statusLabel)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), s.UserID.String())
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), s.ApplicationID.String())
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), s.CreatedAt.Format("2006-01-02 15:04:05"))

		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), centerStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), centerStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), centerStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), centerStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("H%d", row), dataStyle)
		f.SetCellStyle(sheetName, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), centerStyle)

		f.SetRowHeight(sheetName, row, 20)
		row++
	}

	// Atur Lebar Kolom
	colWidths := map[string]float64{
		"A": 6,
		"B": 18,
		"C": 15,
		"D": 14,
		"E": 18,
		"F": 16,
		"G": 38,
		"H": 38,
		"I": 22,
	}
	for col, width := range colWidths {
		f.SetColWidth(sheetName, col, col, width)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
