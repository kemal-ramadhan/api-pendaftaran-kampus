package exporter

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/model"
	"github.com/xuri/excelize/v2"
)

type ExcelExporter interface {
	ExportProgramRecap(recap *model.ProgramRecapResponse, components []model.ScoreComponent) ([]byte, error)
}

type excelExporter struct{}

func NewExcelExporter() ExcelExporter {
	return &excelExporter{}
}

func (e *excelExporter) ExportProgramRecap(recap *model.ProgramRecapResponse, components []model.ScoreComponent) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	sheetName := "Rekap Nilai & Peringkat"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat sheet baru: %w", err)
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	// 1. Judul & Metadata Laporan
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "1F4E79"},
	})
	subTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 10, Color: "595959"},
	})

	_ = f.SetCellValue(sheetName, "A1", "REKAPITULASI NILAI SELEKSI PMB & PEMERINGKATAN PROGRAM STUDI")
	_ = f.SetCellStyle(sheetName, "A1", "A1", titleStyle)

	_ = f.SetCellValue(sheetName, "A3", "Program Studi:")
	_ = f.SetCellValue(sheetName, "B3", recap.StudyProgramCode)
	_ = f.SetCellValue(sheetName, "A4", "Periode ID:")
	_ = f.SetCellValue(sheetName, "B4", recap.PeriodID.String())
	_ = f.SetCellValue(sheetName, "A5", "Jalur ID:")
	_ = f.SetCellValue(sheetName, "B5", recap.PathID.String())
	_ = f.SetCellValue(sheetName, "D3", "Status Penguncian:")
	statusText := "Terbuka (Dapat Diedit)"
	if recap.IsLocked {
		statusText = "TERKUNCI (Final)"
	}
	_ = f.SetCellValue(sheetName, "E3", statusText)
	_ = f.SetCellValue(sheetName, "D4", "Tanggal Ekspor:")
	_ = f.SetCellValue(sheetName, "E4", time.Now().Format("02-01-2006 15:04:05 WIB"))
	_ = f.SetCellValue(sheetName, "D5", "Total Pendaftar:")
	_ = f.SetCellValue(sheetName, "E5", recap.TotalApplicants)

	for row := 3; row <= 5; row++ {
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), subTitleStyle)
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), subTitleStyle)
	}

	// 2. Definisi Header Tabel
	headerRow := 7
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
		},
	})

	headers := []string{"Peringkat", "No. Pendaftaran", "Nama Calon Mahasiswa", "Program Studi"}
	for _, comp := range components {
		headers = append(headers, fmt.Sprintf("%s\n(%.0f%%)", comp.Name, comp.WeightPercent))
	}
	headers = append(headers, "Nilai Akhir\n(Tertimbang)")

	for colIdx, h := range headers {
		cellName, _ := excelize.CoordinatesToCellName(colIdx+1, headerRow)
		_ = f.SetCellValue(sheetName, cellName, h)
		_ = f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
	}
	_ = f.SetRowHeight(sheetName, headerRow, 28)

	// 3. Styling Data Cell
	dataStyleCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})
	dataStyleLeft, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})
	dataStyleNumber, _ := f.NewStyle(&excelize.Style{
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: strPtr("0.00"),
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})
	dataStyleFinalScore, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Color: "000000"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: strPtr("0.00"),
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "top", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
		},
	})

	// 4. Baris Data Pendaftar
	currentRow := headerRow + 1
	for _, p := range recap.Participants {
		// Map score components by code for fast lookup
		scoreMap := make(map[string]float64)
		for _, cs := range p.ComponentScores {
			scoreMap[cs.ComponentCode] = cs.Score
		}

		col := 1
		// Peringkat
		cName, _ := excelize.CoordinatesToCellName(col, currentRow)
		_ = f.SetCellValue(sheetName, cName, p.RankInProgram)
		_ = f.SetCellStyle(sheetName, cName, cName, dataStyleCenter)
		col++

		// Nomor Pendaftaran
		cName, _ = excelize.CoordinatesToCellName(col, currentRow)
		_ = f.SetCellValue(sheetName, cName, p.RegistrationNumber)
		_ = f.SetCellStyle(sheetName, cName, cName, dataStyleCenter)
		col++

		// Nama Calon Mahasiswa
		cName, _ = excelize.CoordinatesToCellName(col, currentRow)
		_ = f.SetCellValue(sheetName, cName, p.FullName)
		_ = f.SetCellStyle(sheetName, cName, cName, dataStyleLeft)
		col++

		// Program Studi
		cName, _ = excelize.CoordinatesToCellName(col, currentRow)
		_ = f.SetCellValue(sheetName, cName, p.StudyProgramCode)
		_ = f.SetCellStyle(sheetName, cName, cName, dataStyleCenter)
		col++

		// Komponen Nilai
		for _, comp := range components {
			cName, _ = excelize.CoordinatesToCellName(col, currentRow)
			val := scoreMap[comp.Code]
			_ = f.SetCellValue(sheetName, cName, val)
			_ = f.SetCellStyle(sheetName, cName, cName, dataStyleNumber)
			col++
		}

		// Nilai Akhir
		cName, _ = excelize.CoordinatesToCellName(col, currentRow)
		_ = f.SetCellValue(sheetName, cName, p.FinalScore)
		_ = f.SetCellStyle(sheetName, cName, cName, dataStyleFinalScore)

		_ = f.SetRowHeight(sheetName, currentRow, 20)
		currentRow++
	}

	// 5. Lebar Kolom
	_ = f.SetColWidth(sheetName, "A", "A", 12)
	_ = f.SetColWidth(sheetName, "B", "B", 18)
	_ = f.SetColWidth(sheetName, "C", "C", 30)
	_ = f.SetColWidth(sheetName, "D", "D", 15)
	lastColLetter, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetColWidth(sheetName, "E", lastColLetter, 18)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("gagal menulis buffer excel: %w", err)
	}

	return buf.Bytes(), nil
}

func strPtr(s string) *string {
	return &s
}
