package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student/internal/service"
)

type StudentController struct {
	svc service.StudentService
}

func NewStudentController(svc service.StudentService) *StudentController {
	return &StudentController{svc: svc}
}

// -----------------------------------------------------------------------------
// Endpoint Calon Mahasiswa (/student/me/reregistration)
// -----------------------------------------------------------------------------

func (ctrl *StudentController) GetMyReregistration(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	detail, err := ctrl.svc.GetMyReregistration(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPassed) {
			response.Error(c, http.StatusUnprocessableEntity, "NOT_PASSED", "Calon mahasiswa belum dinyatakan lulus seleksi atau pendaftaran belum terverifikasi", nil)
			return
		}
		response.InternalServerError(c, "Gagal mengambil data pendaftaran ulang: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data pendaftaran ulang berhasil diambil", detail)
}

func (ctrl *StudentController) UpdateMyReregistration(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	var req model.UpdateReregistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Data permohonan tidak valid: "+err.Error(), nil)
		return
	}

	detail, err := ctrl.svc.UpdateMyReregistration(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrNotPassed) {
			response.Error(c, http.StatusUnprocessableEntity, "NOT_PASSED", "Calon mahasiswa belum dinyatakan lulus seleksi", nil)
			return
		}
		if errors.Is(err, service.ErrAlreadyCompleted) {
			response.Error(c, http.StatusUnprocessableEntity, "ALREADY_COMPLETED", "Pendaftaran ulang sudah berstatus selesai dan tidak dapat diubah", nil)
			return
		}
		response.UnprocessableEntity(c, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Data pendaftaran ulang berhasil diperbarui", detail)
}

func (ctrl *StudentController) CompleteReregistration(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	student, err := ctrl.svc.CompleteReregistration(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotPassed) {
			response.Error(c, http.StatusUnprocessableEntity, "NOT_PASSED", "Calon mahasiswa belum dinyatakan lulus seleksi", nil)
			return
		}
		if errors.Is(err, service.ErrPaymentRequired) {
			response.Error(c, http.StatusUnprocessableEntity, "PAYMENT_REQUIRED", "Tagihan biaya daftar ulang belum lunas atau cicilan termin 1 belum dibayar", nil)
			return
		}
		if errors.Is(err, service.ErrIncompleteData) {
			response.Error(c, http.StatusUnprocessableEntity, "INCOMPLETE_DATA", "Seluruh data tambahan (ukuran jas almamater, kontak darurat, dan pernyataan kepatuhan) wajib diisi lengkap", nil)
			return
		}
		response.InternalServerError(c, "Gagal menyelesaikan proses daftar ulang: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pendaftaran ulang berhasil diselesaikan, Anda telah resmi terdaftar sebagai mahasiswa baru", student)
}

// -----------------------------------------------------------------------------
// Endpoint Administrasi, Rekapitulasi, & Ekspor (/student/admin/students)
// -----------------------------------------------------------------------------

func (ctrl *StudentController) ListStudents(c *gin.Context) {
	roles := gatewayauth.GetUserRoles(c)
	rawScopes := gatewayauth.GetUserScopes(c)
	var scopes []string
	for _, sc := range rawScopes {
		if sc.ID != "" {
			scopes = append(scopes, sc.ID)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	entryYear, _ := strconv.Atoi(c.Query("entry_year"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := model.StudentFilter{
		StudyProgramCode: c.Query("study_program_code"),
		Status:           c.Query("status"),
		EntryYear:        entryYear,
		Search:           c.Query("search"),
		Limit:            perPage,
		Offset:           (page - 1) * perPage,
	}

	students, total, err := ctrl.svc.ListStudents(c.Request.Context(), filter, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProdiAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda hanya berhak mengakses data mahasiswa pada program studi yang menjadi wewenang Anda")
			return
		}
		response.InternalServerError(c, "Gagal mengambil daftar mahasiswa: "+err.Error())
		return
	}

	response.SuccessWithMeta(c, http.StatusOK, "Daftar mahasiswa berhasil diambil", students, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

func (ctrl *StudentController) ExportStudentsXLSX(c *gin.Context) {
	roles := gatewayauth.GetUserRoles(c)
	rawScopes := gatewayauth.GetUserScopes(c)
	var scopes []string
	for _, sc := range rawScopes {
		if sc.ID != "" {
			scopes = append(scopes, sc.ID)
		}
	}

	entryYear, _ := strconv.Atoi(c.Query("entry_year"))

	filter := model.StudentFilter{
		StudyProgramCode: c.Query("study_program_code"),
		Status:           c.Query("status"),
		EntryYear:        entryYear,
		Search:           c.Query("search"),
	}

	xlsxBytes, err := ctrl.svc.ExportStudentsXLSX(c.Request.Context(), filter, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProdiAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda hanya berhak mengekspor data mahasiswa pada program studi Anda")
			return
		}
		response.InternalServerError(c, "Gagal mengekspor data mahasiswa: "+err.Error())
		return
	}

	fileName := fmt.Sprintf("Laporan_Mahasiswa_Baru_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
}

// -----------------------------------------------------------------------------
// Endpoint Internal Inter-Service (/internal/students/...)
// -----------------------------------------------------------------------------

func (ctrl *StudentController) InternalGetStudentByApplicationID(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	st, err := ctrl.svc.GetStudentByApplicationID(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(c, "Data mahasiswa belum terdaftar untuk pendaftaran ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data mahasiswa: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data mahasiswa berhasil diambil", st)
}

type UpdateNIMInput struct {
	NIM string `json:"nim" binding:"required"`
}

func (ctrl *StudentController) InternalUpdateNIM(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	var input UpdateNIMInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data NIM wajib diisi: "+err.Error())
		return
	}

	if err := ctrl.svc.UpdateStudentNIM(c.Request.Context(), appID, input.NIM); err != nil {
		response.InternalServerError(c, "Gagal memperbarui NIM mahasiswa: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "NIM mahasiswa berhasil diperbarui", gin.H{
		"application_id": appID,
		"nim":            input.NIM,
		"status":         model.StudentStatusActive,
	})
}

func (ctrl *StudentController) ProcessOutbox(c *gin.Context) {
	count, err := ctrl.svc.ProcessPendingOutboxEvents(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal memproses antrean outbox: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Antrean outbox berhasil diproses", gin.H{
		"processed_count": count,
	})
}

func (ctrl *StudentController) InternalWithdraw(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	if err := ctrl.svc.WithdrawApplication(c.Request.Context(), appID); err != nil {
		response.InternalServerError(c, "Gagal membatalkan daftar ulang mahasiswa: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pengunduran diri pendaftaran ulang berhasil diproses", gin.H{
		"application_id": appID,
		"withdrawn":      true,
	})
}

// -----------------------------------------------------------------------------
// Endpoint Administrasi Enrolment & Penetapan NIM (Butir h)
// -----------------------------------------------------------------------------

func (ctrl *StudentController) GetEnrolmentList(c *gin.Context) {
	roles := gatewayauth.GetUserRoles(c)
	rawScopes := gatewayauth.GetUserScopes(c)
	var scopes []string
	for _, sc := range rawScopes {
		if sc.ID != "" {
			scopes = append(scopes, sc.ID)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := model.EnrolmentFilter{
		PeriodID:         c.Query("period_id"),
		Filter:           c.DefaultQuery("filter", "all"),
		Search:           c.Query("q"),
		StudyProgramCode: c.Query("study_program_code"),
		Limit:            perPage,
		Offset:           (page - 1) * perPage,
	}

	items, total, err := ctrl.svc.GetEnrolmentList(c.Request.Context(), filter, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProdiAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil daftar enrolment: "+err.Error())
		return
	}

	response.SuccessWithMeta(c, http.StatusOK, "Daftar enrolment mahasiswa berhasil diambil", items, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

func (ctrl *StudentController) IssueNIMManual(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID mahasiswa tidak valid")
		return
	}

	st, err := ctrl.svc.IssueNIMManual(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrStudentNotFound) {
			response.NotFound(c, "Data mahasiswa tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrPaymentRequired) {
			response.Error(c, http.StatusUnprocessableEntity, "PAYMENT_REQUIRED", "Tagihan UKT belum lunas", nil)
			return
		}
		response.InternalServerError(c, "Gagal menetapkan NIM: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "NIM berhasil ditetapkan", st)
}

func (ctrl *StudentController) IssueNIMBatch(c *gin.Context) {
	var req model.BatchNIMRequest
	_ = c.ShouldBindJSON(&req)

	resp, err := ctrl.svc.IssueNIMBatch(c.Request.Context(), req)
	if err != nil {
		response.InternalServerError(c, "Gagal memproses penerbitan NIM batch: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Proses penerbitan NIM massal berhasil dijalankan", resp)
}

// -----------------------------------------------------------------------------
// Endpoint Gugus PKKMB (Butir h)
// -----------------------------------------------------------------------------

func (ctrl *StudentController) ListPKKMBGroups(c *gin.Context) {
	var periodID *uuid.UUID
	periodStr := c.Query("period_id")
	if periodStr != "" {
		if pid, err := uuid.Parse(periodStr); err == nil {
			periodID = &pid
		}
	}

	groups, err := ctrl.svc.ListPKKMBGroups(c.Request.Context(), periodID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar gugus PKKMB berhasil diambil", groups)
}

func (ctrl *StudentController) CreatePKKMBGroup(c *gin.Context) {
	var req model.CreatePKKMBGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data gugus PKKMB tidak valid: "+err.Error())
		return
	}

	group, err := ctrl.svc.CreatePKKMBGroup(c.Request.Context(), &req)
	if err != nil {
		response.InternalServerError(c, "Gagal membuat gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Gugus PKKMB baru berhasil dibuat", group)
}

func (ctrl *StudentController) UpdatePKKMBGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID gugus tidak valid")
		return
	}

	var req model.UpdatePKKMBGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data pembaruan tidak valid: "+err.Error())
		return
	}

	group, err := ctrl.svc.UpdatePKKMBGroup(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrPKKMBGroupNotFound) {
			response.NotFound(c, "Gugus PKKMB tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memperbarui gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Gugus PKKMB berhasil diperbarui", group)
}

func (ctrl *StudentController) DeletePKKMBGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID gugus tidak valid")
		return
	}

	if err := ctrl.svc.DeletePKKMBGroup(c.Request.Context(), id); err != nil {
		response.InternalServerError(c, "Gagal menghapus gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Gugus PKKMB berhasil dihapus", nil)
}

func (ctrl *StudentController) AutoAssignPKKMB(c *gin.Context) {
	var req model.AutoAssignPKKMBRequest
	_ = c.ShouldBindJSON(&req)

	res, err := ctrl.svc.AutoAssignPKKMB(c.Request.Context(), &req)
	if err != nil {
		response.InternalServerError(c, "Gagal melakukan penugasan otomatis gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Penugasan otomatis gugus PKKMB berhasil", res)
}

func (ctrl *StudentController) AssignPKKMBManual(c *gin.Context) {
	studentIDStr := c.Param("student_id")
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format student_id tidak valid")
		return
	}

	var req model.AssignPKKMBStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data penugasan tidak valid: "+err.Error())
		return
	}

	var adminID *uuid.UUID
	if uid, errU := gatewayauth.GetUserID(c); errU == nil {
		adminID = &uid
	}

	if err := ctrl.svc.AssignPKKMBManual(c.Request.Context(), studentID, req.GroupID, adminID); err != nil {
		if errors.Is(err, service.ErrPKKMBGroupNotFound) {
			response.NotFound(c, "Gugus PKKMB tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal menugaskan mahasiswa ke gugus PKKMB: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Mahasiswa berhasil ditugaskan ke gugus PKKMB", nil)
}

func (ctrl *StudentController) ExportPKKMB(c *gin.Context) {
	var periodID *uuid.UUID
	periodStr := c.Query("period_id")
	if periodStr != "" {
		if pid, err := uuid.Parse(periodStr); err == nil {
			periodID = &pid
		}
	}

	data, err := ctrl.svc.ExportPKKMBXLSX(c.Request.Context(), periodID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data gugus PKKMB: "+err.Error())
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=gugus_pkkmb.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (ctrl *StudentController) ExportSIAKAD(c *gin.Context) {
	format := strings.ToLower(c.DefaultQuery("format", "csv"))

	data, err := ctrl.svc.ExportSIAKAD(c.Request.Context(), format)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data SIAKAD: "+err.Error())
		return
	}

	if format == "xlsx" {
		c.Header("Content-Disposition", "attachment; filename=siakad_export.xlsx")
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=siakad_export.csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", data)
}

