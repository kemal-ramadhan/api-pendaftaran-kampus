package exporter

import (
	"bytes"
	"fmt"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"github.com/xuri/excelize/v2"
)

// GenerateApplicationsExcel menghasilkan binary stream file .xlsx dari daftar aplikasi pendaftar
func GenerateApplicationsExcel(apps []model.Application) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Pendaftar PMB"
	f.SetSheetName("Sheet1", sheetName)

	// Header Kolom
	headers := []string{
		"No",
		"No. Registrasi",
		"User ID",
		"Gelombang",
		"Jalur Masuk",
		"Status",
		"Pilihan 1 (Prodi ID)",
		"Pilihan 2 (Prodi ID)",
		"Waktu Pendaftaran",
		"Waktu Submit",
		"Waktu Verifikasi",
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9E1F2"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat style header: %w", err)
	}

	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, header)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	for rowIdx, app := range apps {
		row := rowIdx + 2

		periodName := ""
		if app.Period != nil {
			periodName = app.Period.Name
		}
		pathName := ""
		if app.AdmissionPath != nil {
			pathName = app.AdmissionPath.Name
		}

		choice2Str := "-"
		if app.Choice2ID != nil {
			choice2Str = app.Choice2ID.String()
		}

		submittedAtStr := "-"
		if app.SubmittedAt != nil {
			submittedAtStr = app.SubmittedAt.Format("02-01-2006 15:04:05")
		}

		verifiedAtStr := "-"
		if app.VerifiedAt != nil {
			verifiedAtStr = app.VerifiedAt.Format("02-01-2006 15:04:05")
		}

		values := []interface{}{
			rowIdx + 1,
			app.RegistrationNo,
			app.UserID.String(),
			periodName,
			pathName,
			string(app.Status),
			app.Choice1ID.String(),
			choice2Str,
			app.CreatedAt.Format("02-01-2006 15:04:05"),
			submittedAtStr,
			verifiedAtStr,
		}

		for colIdx, val := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	// Atur lebar kolom
	_ = f.SetColWidth(sheetName, "A", "A", 6)
	_ = f.SetColWidth(sheetName, "B", "B", 24)
	_ = f.SetColWidth(sheetName, "C", "C", 38)
	_ = f.SetColWidth(sheetName, "D", "D", 22)
	_ = f.SetColWidth(sheetName, "E", "E", 18)
	_ = f.SetColWidth(sheetName, "F", "F", 20)
	_ = f.SetColWidth(sheetName, "G", "H", 38)
	_ = f.SetColWidth(sheetName, "I", "K", 22)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("gagal menulis buffer excel: %w", err)
	}

	return buf.Bytes(), nil
}
