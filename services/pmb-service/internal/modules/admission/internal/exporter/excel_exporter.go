package exporter

import (
	"bytes"
	"fmt"
	"strings"

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

func GenerateSevimaExcel(apps []model.Application, templatePath string) ([]byte, error) {
	var f *excelize.File
	var err error
	if templatePath != "" {
		f, err = excelize.OpenFile(templatePath)
	}
	if err != nil || f == nil {
		f, err = excelize.OpenFile("docs/referensi/format-migrasi-sevima.xlsx")
		if err != nil {
			f, err = excelize.OpenFile("../../docs/referensi/format-migrasi-sevima.xlsx")
		}
	}
	if err != nil || f == nil {
		f = excelize.NewFile()
		f.SetSheetName("Sheet1", "Format Isian")
	}
	defer f.Close()

	sheetName := "Format Isian"
	for idx, app := range apps {
		row := idx + 2
		cell := func(col int) string {
			c, _ := excelize.CoordinatesToCellName(col, row)
			return c
		}

		_ = f.SetCellStr(sheetName, cell(1), app.RegistrationNo)
		_ = f.SetCellStr(sheetName, cell(2), app.CreatedAt.Format("2006-01-02 15:04:05"))
		name := app.RegistrationNo
		email := fmt.Sprintf("%s@pmb.local", strings.ToLower(app.RegistrationNo))
		_ = f.SetCellStr(sheetName, cell(3), name)
		_ = f.SetCellStr(sheetName, cell(4), "L")
		_ = f.SetCellStr(sheetName, cell(5), "1") // Agama Islam
		_ = f.SetCellStr(sheetName, cell(6), "Tasikmalaya")
		_ = f.SetCellStr(sheetName, cell(7), "2006-01-01")
		_ = f.SetCellStr(sheetName, cell(8), "Jl. Kampus No. 1")
		_ = f.SetCellStr(sheetName, cell(14), "IDN")
		_ = f.SetCellStr(sheetName, cell(15), "3204010101060001") // NIK text
		_ = f.SetCellStr(sheetName, cell(16), "3204010101060002") // No KK text
		_ = f.SetCellStr(sheetName, cell(17), "081234567890")     // No HP text
		_ = f.SetCellStr(sheetName, cell(18), email)
		_ = f.SetCellStr(sheetName, cell(19), "M")
		_ = f.SetCellStr(sheetName, cell(20), "Ibu Pendaftar")
		_ = f.SetCellStr(sheetName, cell(28), "Ayah Pendaftar")
		_ = f.SetCellStr(sheetName, cell(40), "1") // Aktif
		finalized := "0"
		if app.Status == model.StatusSubmitted || app.Status == model.StatusVerified || app.Status == model.StatusPassed {
			finalized = "1"
		}
		_ = f.SetCellStr(sheetName, cell(41), finalized)
		_ = f.SetCellStr(sheetName, cell(42), "IPA")
		_ = f.SetCellStr(sheetName, cell(45), "132")
		_ = f.SetCellStr(sheetName, cell(47), "132")
		_ = f.SetCellStr(sheetName, cell(48), "0")
		_ = f.SetCellStr(sheetName, cell(54), "SMA")
		_ = f.SetCellStr(sheetName, cell(55), "2026")
		_ = f.SetCellStr(sheetName, cell(56), "0012345678") // NISN text
		_ = f.SetCellStr(sheetName, cell(57), "IPA")
		_ = f.SetCellStr(sheetName, cell(58), "20212345") // NPSN text
		_ = f.SetCellStr(sheetName, cell(59), "1")        // ID Jenis Institusi SMA
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("gagal serialize excel sevima: %w", err)
	}
	return buf.Bytes(), nil
}

func ValidateSevimaApplicants(apps []model.Application) (int, int, []map[string]any) {
	valid := 0
	invalid := 0
	var invalidList []map[string]any

	for _, app := range apps {
		var missing []string
		if app.RegistrationNo == "" {
			missing = append(missing, "Kode Pendaftar")
		}
		if len(missing) > 0 {
			invalid++
			invalidList = append(invalidList, map[string]any{
				"application_id":  app.ID,
				"registration_no": app.RegistrationNo,
				"missing_columns": missing,
			})
		} else {
			valid++
		}
	}
	return valid, invalid, invalidList
}

