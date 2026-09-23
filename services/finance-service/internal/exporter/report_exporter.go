package exporter

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/xuri/excelize/v2"
)

type ReportExporter interface {
	ExportFinancialReport(summary *model.FinancialReportSummaryResponse, invoices []model.Invoice) ([]byte, error)
}

type reportExporter struct{}

func NewReportExporter() ReportExporter {
	return &reportExporter{}
}

func (e *reportExporter) ExportFinancialReport(summary *model.FinancialReportSummaryResponse, invoices []model.Invoice) ([]byte, error) {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	// ----------------------------------------------------
	// Sheet 1: Ringkasan Eksekutif
	// ----------------------------------------------------
	sheetSummary := "Ringkasan Keuangan"
	idxSummary, err := f.NewSheet(sheetSummary)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat sheet ringkasan: %w", err)
	}
	f.SetActiveSheet(idxSummary)
	_ = f.DeleteSheet("Sheet1")

	// Styling
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "1F4E79"},
	})
	subTitleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 10, Color: "595959"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
		},
	})
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
		},
	})
	numberStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: func() *string {
			s := "#,##0"
			return &s
		}(),
		Border: []excelize.Border{
			{Type: "left", Color: "D9D9D9", Style: 1},
			{Type: "top", Color: "D9D9D9", Style: 1},
			{Type: "bottom", Color: "D9D9D9", Style: 1},
			{Type: "right", Color: "D9D9D9", Style: 1},
		},
	})

	_ = f.SetCellValue(sheetSummary, "A1", "LAPORAN RINGKASAN PENERIMAAN KEUANGAN PMB")
	_ = f.SetCellStyle(sheetSummary, "A1", "A1", titleStyle)

	_ = f.SetCellValue(sheetSummary, "A2", fmt.Sprintf("Waktu Cetak: %s", time.Now().Format("02-01-2006 15:04:05 WIB")))
	_ = f.SetCellStyle(sheetSummary, "A2", "A2", subTitleStyle)

	// Ringkasan Indikator Utama
	_ = f.SetCellValue(sheetSummary, "A4", "METRIK KEUANGAN")
	_ = f.SetCellValue(sheetSummary, "B4", "NILAI (RUPIAH / JUMLAH)")
	_ = f.SetCellStyle(sheetSummary, "A4", "B4", headerStyle)

	metrics := []struct {
		Label string
		Value int64
		IsNum bool
	}{
		{"Total Tagihan Diterbitkan", summary.TotalInvoices, false},
		{"Total Nilai Tagihan (Expected)", summary.TotalExpectedAmount, true},
		{"Total Pembayaran Masuk (Paid)", summary.TotalPaidAmount, true},
		{"Total Sisa Tunggakan (Outstanding)", summary.TotalOutstandingAmount, true},
	}

	for idx, m := range metrics {
		row := 5 + idx
		cellA := fmt.Sprintf("A%d", row)
		cellB := fmt.Sprintf("B%d", row)
		_ = f.SetCellValue(sheetSummary, cellA, m.Label)
		_ = f.SetCellValue(sheetSummary, cellB, m.Value)
		_ = f.SetCellStyle(sheetSummary, cellA, cellA, dataStyle)
		if m.IsNum {
			_ = f.SetCellStyle(sheetSummary, cellB, cellB, numberStyle)
		} else {
			_ = f.SetCellStyle(sheetSummary, cellB, cellB, dataStyle)
		}
	}

	// Ringkasan per Jenis Biaya
	rowFTStart := 11
	_ = f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", rowFTStart), "RINCIAN PER JENIS BIAYA")
	_ = f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", rowFTStart), fmt.Sprintf("A%d", rowFTStart), titleStyle)

	headersFT := []string{"Jenis Biaya", "Jumlah Tagihan", "Total Tagihan", "Terbayar", "Tunggakan"}
	colsFT := []string{"A", "B", "C", "D", "E"}
	for i, h := range headersFT {
		cell := fmt.Sprintf("%s%d", colsFT[i], rowFTStart+1)
		_ = f.SetCellValue(sheetSummary, cell, h)
		_ = f.SetCellStyle(sheetSummary, cell, cell, headerStyle)
	}

	for i, item := range summary.ByFeeType {
		row := rowFTStart + 2 + i
		_ = f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", row), item.FeeType)
		_ = f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", row), item.Count)
		_ = f.SetCellValue(sheetSummary, fmt.Sprintf("C%d", row), item.TotalAmount)
		_ = f.SetCellValue(sheetSummary, fmt.Sprintf("D%d", row), item.PaidAmount)
		_ = f.SetCellValue(sheetSummary, fmt.Sprintf("E%d", row), item.OutstandingAmount)

		_ = f.SetCellStyle(sheetSummary, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)
		_ = f.SetCellStyle(sheetSummary, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataStyle)
		_ = f.SetCellStyle(sheetSummary, fmt.Sprintf("C%d", row), fmt.Sprintf("E%d", row), numberStyle)
	}

	_ = f.SetColWidth(sheetSummary, "A", "A", 35)
	_ = f.SetColWidth(sheetSummary, "B", "B", 25)
	_ = f.SetColWidth(sheetSummary, "C", "E", 20)

	// ----------------------------------------------------
	// Sheet 2: Rincian Tagihan & Cicilan
	// ----------------------------------------------------
	sheetDetail := "Rincian Tagihan"
	_, _ = f.NewSheet(sheetDetail)

	_ = f.SetCellValue(sheetDetail, "A1", "DATA RINCIAN TAGIHAN & PEMBAYARAN")
	_ = f.SetCellStyle(sheetDetail, "A1", "A1", titleStyle)

	headersDetail := []string{
		"No", "Nomor Tagihan", "Jenis Biaya", "Skema", "Total Tagihan",
		"Terbayar", "Sisa", "Status", "Jatuh Tempo", "Info Termin",
	}
	colsDetail := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for i, h := range headersDetail {
		cell := fmt.Sprintf("%s3", colsDetail[i])
		_ = f.SetCellValue(sheetDetail, cell, h)
		_ = f.SetCellStyle(sheetDetail, cell, cell, headerStyle)
	}

	for idx, inv := range invoices {
		row := 4 + idx
		outstanding := inv.Amount - inv.PaidAmount
		if outstanding < 0 {
			outstanding = 0
		}

		terminInfo := "-"
		if inv.PaymentScheme == "installment" && len(inv.Installments) > 0 {
			paidCount := 0
			for _, inst := range inv.Installments {
				if inst.Status == model.InstallmentStatusPaid {
					paidCount++
				}
			}
			terminInfo = fmt.Sprintf("%d/%d Termin Lunas", paidCount, len(inv.Installments))
		}

		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("A%d", row), idx+1)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("B%d", row), inv.InvoiceNumber)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("C%d", row), inv.FeeType)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("D%d", row), inv.PaymentScheme)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("E%d", row), inv.Amount)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("F%d", row), inv.PaidAmount)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("G%d", row), outstanding)
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("H%d", row), string(inv.Status))
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("I%d", row), inv.DueDate.Format("02-01-2006"))
		_ = f.SetCellValue(sheetDetail, fmt.Sprintf("J%d", row), terminInfo)

		_ = f.SetCellStyle(sheetDetail, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), dataStyle)
		_ = f.SetCellStyle(sheetDetail, fmt.Sprintf("E%d", row), fmt.Sprintf("G%d", row), numberStyle)
		_ = f.SetCellStyle(sheetDetail, fmt.Sprintf("H%d", row), fmt.Sprintf("J%d", row), dataStyle)
	}

	_ = f.SetColWidth(sheetDetail, "A", "A", 6)
	_ = f.SetColWidth(sheetDetail, "B", "B", 22)
	_ = f.SetColWidth(sheetDetail, "C", "D", 16)
	_ = f.SetColWidth(sheetDetail, "E", "G", 18)
	_ = f.SetColWidth(sheetDetail, "H", "I", 15)
	_ = f.SetColWidth(sheetDetail, "J", "J", 20)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("gagal menulis file excel: %w", err)
	}

	return buf.Bytes(), nil
}
