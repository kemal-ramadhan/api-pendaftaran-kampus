package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type ReportController struct {
	reportSvc      service.ReportService
	installmentSvc service.InstallmentService
}

func NewReportController(reportSvc service.ReportService, installmentSvc service.InstallmentService) *ReportController {
	return &ReportController{
		reportSvc:      reportSvc,
		installmentSvc: installmentSvc,
	}
}

// GetSummary mengambil ringkasan laporan penerimaan keuangan dan tunggakan
func (ctrl *ReportController) GetSummary(c *gin.Context) {
	summary, err := ctrl.reportSvc.GetFinancialReportSummary(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil ringkasan laporan keuangan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Ringkasan laporan keuangan berhasil diambil", summary)
}

// ExportXLSX memproduksi berkas spreadsheet resmi Excel .xlsx untuk laporan keuangan
func (ctrl *ReportController) ExportXLSX(c *gin.Context) {
	xlsxBytes, err := ctrl.reportSvc.ExportFinancialReportXLSX(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor laporan keuangan: "+err.Error())
		return
	}

	filename := fmt.Sprintf("laporan-keuangan-pmb-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
}

// ProcessDueDatesCron memicu proses pengingat jatuh tempo H-3, H-1, dan penandaan expired
func (ctrl *ReportController) ProcessDueDatesCron(c *gin.Context) {
	remindersSent, expiredMarked, err := ctrl.installmentSvc.ProcessDueRemindersAndExpirations(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal memproses pemantauan jatuh tempo: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pemrosesan pengingat jatuh tempo dan status kedaluwarsa berhasil", gin.H{
		"reminders_sent": remindersSent,
		"expired_marked": expiredMarked,
	})
}
