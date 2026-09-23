package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type AdminController struct {
	svc service.AdmissionService
}

func NewAdminController(svc service.AdmissionService) *AdminController {
	return &AdminController{svc: svc}
}

type VerifyDocumentsInput struct {
	Reviews []service.DocumentReviewItem `json:"reviews" binding:"required"`
}

func (ctrl *AdminController) ListApplications(c *gin.Context) {
	roles := gatewayauth.GetUserRoles(c)
	isProdiOnly := gatewayauth.HasRole(c, "prodi") && !gatewayauth.HasAnyRole(c, "superadmin", "admin_pmb")

	var prodiUUID *uuid.UUID

	if isProdiOnly {
		prodiIDStr, ok := gatewayauth.GetScopeID(c, "prodi", "prodi")
		if !ok || prodiIDStr == "" {
			prodiIDStr = c.GetHeader("X-User-Prodi-ID")
		}
		if prodiIDStr == "" {
			prodiIDStr = c.Query("prodi_id")
		}
		if prodiIDStr == "" {
			response.Forbidden(c, response.ErrCodeForbidden, "Akses ditolak: role prodi wajib memiliki scope program studi")
			return
		}
		parsedProdi, err := uuid.Parse(prodiIDStr)
		if err != nil {
			response.BadRequest(c, "INVALID_PRODI_ID", "Scope program studi tidak valid")
			return
		}
		prodiUUID = &parsedProdi
	}

	filter := repository.ApplicationFilter{
		StudyProgramID: prodiUUID,
		Search:         c.Query("search"),
	}

	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			filter.PeriodID = &u
		}
	}

	if pathIDStr := c.Query("path_id"); pathIDStr != "" {
		if pathID, err := strconv.ParseUint(pathIDStr, 10, 32); err == nil {
			u := uint(pathID)
			filter.AdmissionPathID = &u
		}
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	limit := 10
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	filter.Limit = limit

	page := 1
	if pgStr := c.Query("page"); pgStr != "" {
		if pg, err := strconv.Atoi(pgStr); err == nil && pg > 0 {
			page = pg
		}
	}
	filter.Offset = (page - 1) * limit

	apps, total, err := ctrl.svc.ListApplicationsAdmin(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar aplikasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar aplikasi pendaftaran berhasil diambil", gin.H{
		"items": apps,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"roles": roles,
		},
	})
}

func (ctrl *AdminController) GetApplicationDetail(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	role := "admin_pmb"
	if len(roles) > 0 {
		role = roles[0]
	}

	userID, _ := gatewayauth.GetUserID(c)

	var prodiUUID *uuid.UUID
	if gatewayauth.HasRole(c, "prodi") {
		prodiIDStr, ok := gatewayauth.GetScopeID(c, "prodi", "prodi")
		if !ok || prodiIDStr == "" {
			prodiIDStr = c.GetHeader("X-User-Prodi-ID")
		}
		if prodiIDStr == "" {
			prodiIDStr = c.Query("prodi_id")
		}
		if prodiIDStr != "" {
			if parsed, err := uuid.Parse(prodiIDStr); err == nil {
				prodiUUID = &parsed
			}
		}
	}

	app, err := ctrl.svc.GetApplicationDetail(c.Request.Context(), appID, userID, role, prodiUUID)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Akses ditolak: Anda tidak memiliki akses ke berkas pendaftar ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail pendaftaran berhasil diambil", app)
}

// VerifyDocuments memverifikasi setiap berkas pendaftar dan memperbarui status pendaftaran
func (ctrl *AdminController) VerifyDocuments(c *gin.Context) {
	reviewerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas admin tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var input VerifyDocumentsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data verifikasi berkas tidak valid: "+err.Error())
		return
	}

	app, err := ctrl.svc.VerifyApplicationDocuments(c.Request.Context(), appID, reviewerID, input.Reviews)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrApplicationNotSubmitted) {
			response.BadRequest(c, "INVALID_STATUS", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memproses verifikasi berkas: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Verifikasi berkas berhasil diproses", app)
}

// GetDocumentURL mengambil presigned URL berkas pendaftar untuk admin (berlaku 300 detik / 5 menit)
func (ctrl *AdminController) GetDocumentURL(c *gin.Context) {
	reviewerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas admin tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	reqCode := c.Param("code")
	roles := gatewayauth.GetUserRoles(c)
	role := "admin_pmb"
	if len(roles) > 0 {
		role = roles[0]
	}

	url, err := ctrl.svc.GetDocumentPresignedURL(c.Request.Context(), appID, reqCode, reviewerID, role)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) || errors.Is(err, repository.ErrDocumentNotFound) {
			response.NotFound(c, "Dokumen atau pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil URL dokumen: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Presigned URL dokumen berhasil dibuat (berlaku 5 menit)", gin.H{"url": url})
}

// ExportApplications mengekspor seluruh data pendaftar ke spreadsheet Excel (.xlsx)
func (ctrl *AdminController) ExportApplications(c *gin.Context) {
	filter := repository.ApplicationFilter{
		Search: c.Query("search"),
	}

	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			filter.PeriodID = &u
		}
	}

	if pathIDStr := c.Query("path_id"); pathIDStr != "" {
		if pathID, err := strconv.ParseUint(pathIDStr, 10, 32); err == nil {
			u := uint(pathID)
			filter.AdmissionPathID = &u
		}
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	if prodiIDStr := c.Query("prodi_id"); prodiIDStr != "" {
		if pUUID, err := uuid.Parse(prodiIDStr); err == nil {
			filter.StudyProgramID = &pUUID
		}
	}

	xlsxBytes, err := ctrl.svc.ExportApplicationsXLSX(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data pendaftaran ke Excel: "+err.Error())
		return
	}

	fileName := fmt.Sprintf("pmb_applications_export_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Length", strconv.Itoa(len(xlsxBytes)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
}

// GetStats mengembalikan ringkasan statistik pendaftaran PMB
func (ctrl *AdminController) GetStats(c *gin.Context) {
	stats, err := ctrl.svc.GetStats(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data statistik pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Statistik pendaftaran berhasil diambil", stats)
}
