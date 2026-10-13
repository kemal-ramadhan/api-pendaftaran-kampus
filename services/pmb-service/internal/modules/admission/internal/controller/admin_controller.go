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
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/service"
)

type AdminController struct {
	svc       service.AdmissionService
	periodRepo repository.PeriodRepository
	pathRepo   repository.AdmissionPathRepository
	quotaRepo  repository.QuotaRepository
	reqRepo    repository.DocumentRequirementRepository
}

func NewAdminController(
	svc service.AdmissionService,
	periodRepo repository.PeriodRepository,
	pathRepo repository.AdmissionPathRepository,
	quotaRepo repository.QuotaRepository,
	reqRepo repository.DocumentRequirementRepository,
) *AdminController {
	return &AdminController{
		svc:        svc,
		periodRepo: periodRepo,
		pathRepo:   pathRepo,
		quotaRepo:  quotaRepo,
		reqRepo:    reqRepo,
	}
}

type VerifyDocumentsInput struct {
	Reviews []service.DocumentReviewItem `json:"reviews"`
	Status  string                       `json:"status,omitempty"`
	Notes   string                       `json:"notes,omitempty"`
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

// ExportSevimaApplications mengekspor data pendaftar format SEVIMA (62 kolom)
func (ctrl *AdminController) ExportSevimaApplications(c *gin.Context) {
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

	if prodiIDStr := c.Query("program_id"); prodiIDStr != "" {
		if pUUID, err := uuid.Parse(prodiIDStr); err == nil {
			filter.StudyProgramID = &pUUID
		}
	}

	xlsxBytes, err := ctrl.svc.ExportSevimaXLSX(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data SEVIMA ke Excel: "+err.Error())
		return
	}

	fileName := fmt.Sprintf("format-migrasi-sevima_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Length", strconv.Itoa(len(xlsxBytes)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
}

// ValidateSevimaApplications memvalidasi data pendaftar untuk ekspor SEVIMA
func (ctrl *AdminController) ValidateSevimaApplications(c *gin.Context) {
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	res, err := ctrl.svc.ValidateSevima(c.Request.Context(), periodID)
	if err != nil {
		response.InternalServerError(c, "Gagal memvalidasi data pendaftaran SEVIMA: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Hasil validasi migrasi SEVIMA berhasil diambil", res)
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

// Master Data Handlers (PRD C.4.4 / TINGGI-1)

func (ctrl *AdminController) ListPeriods(c *gin.Context) {
	periods, err := ctrl.periodRepo.FindAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar gelombang pendaftaran: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar gelombang berhasil diambil", periods)
}

func (ctrl *AdminController) CreatePeriod(c *gin.Context) {
	var input model.Period
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data gelombang tidak valid: "+err.Error())
		return
	}
	if err := ctrl.periodRepo.Create(c.Request.Context(), &input); err != nil {
		response.InternalServerError(c, "Gagal membuat gelombang: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Gelombang pendaftaran berhasil dibuat", input)
}

func (ctrl *AdminController) ListPaths(c *gin.Context) {
	paths, err := ctrl.pathRepo.FindAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar jalur pendaftaran: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar jalur pendaftaran berhasil diambil", paths)
}

func (ctrl *AdminController) CreatePath(c *gin.Context) {
	var input model.AdmissionPath
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data jalur pendaftaran tidak valid: "+err.Error())
		return
	}
	if err := ctrl.pathRepo.Create(c.Request.Context(), &input); err != nil {
		response.InternalServerError(c, "Gagal membuat jalur pendaftaran: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Jalur pendaftaran berhasil dibuat", input)
}

func (ctrl *AdminController) ListQuotas(c *gin.Context) {
	quotas, err := ctrl.quotaRepo.FindAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar kuota: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar kuota berhasil diambil", quotas)
}

func (ctrl *AdminController) CreateQuota(c *gin.Context) {
	var input model.Quota
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data kuota tidak valid: "+err.Error())
		return
	}
	if err := ctrl.quotaRepo.Create(c.Request.Context(), &input); err != nil {
		response.InternalServerError(c, "Gagal membuat kuota: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Kuota berhasil dibuat", input)
}

func (ctrl *AdminController) ListRequirements(c *gin.Context) {
	reqs, err := ctrl.reqRepo.FindAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar syarat berkas: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar persyaratan dokumen berhasil diambil", reqs)
}

func (ctrl *AdminController) CreateRequirement(c *gin.Context) {
	var input model.DocumentRequirement
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data syarat berkas tidak valid: "+err.Error())
		return
	}
	if err := ctrl.reqRepo.Create(c.Request.Context(), &input); err != nil {
		response.InternalServerError(c, "Gagal membuat persyaratan dokumen: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Persyaratan dokumen berhasil dibuat", input)
}

// GetHeaderSummary menangani GET /admission/admin/header-summary
func (ctrl *AdminController) GetHeaderSummary(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	res, err := ctrl.svc.GetHeaderSummary(c.Request.Context(), periodID, scope)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil ringkasan header: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Ringkasan header pendaftaran berhasil diambil", res)
}

// GetOverview menangani GET /admission/admin/overview
func (ctrl *AdminController) GetOverview(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	roles := gatewayauth.GetUserRoles(c)
	role := "admin_pmb"
	if len(roles) > 0 {
		role = roles[0]
	}

	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	res, err := ctrl.svc.GetOverview(c.Request.Context(), periodID, scope, role)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat overview eksekutif: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Overview pendaftaran berhasil diambil", res)
}

// GetDocumentQueue menangani GET /admission/admin/document-queue
func (ctrl *AdminController) GetDocumentQueue(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	status := c.Query("status")
	q := c.Query("q")
	if q == "" {
		q = c.Query("search")
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := ctrl.svc.GetDocumentQueue(c.Request.Context(), periodID, status, q, page, limit, scope)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat antrean berkas: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Antrean berkas pendaftaran berhasil diambil", res)
}

// GetApplicationDocuments menangani GET /admission/admin/applications/:id/documents
func (ctrl *AdminController) GetApplicationDocuments(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	res, err := ctrl.svc.GetApplicationDocumentsModal(c.Request.Context(), appID, scope)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memuat dokumen pendaftaran: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Dokumen pendaftaran berhasil diambil", res)
}

// GetInstitutionDashboard menangani GET /admission/admin/institution-dashboard
func (ctrl *AdminController) GetInstitutionDashboard(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}
	level := c.Query("level")
	code := c.Query("code")

	res, err := ctrl.svc.GetInstitutionDashboard(c.Request.Context(), periodID, level, code, scope)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat dashboard institusi: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Dashboard institusi berhasil diambil", res)
}

// GetReportsPreview menangani GET /admission/admin/reports/preview
func (ctrl *AdminController) GetReportsPreview(c *gin.Context) {
	scope := rbac.ScopeFilter(c)
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}
	program := c.Query("program")

	res, err := ctrl.svc.GetReportsPreview(c.Request.Context(), periodID, program, scope)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat pratinjau laporan: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Pratinjau laporan berhasil diambil", res)
}

// PrintOfficialReport menangani GET /admission/admin/reports/:type/official.pdf dan /reports/official.pdf
func (ctrl *AdminController) PrintOfficialReport(c *gin.Context) {
	docType := c.Param("type")
	if docType == "" {
		docType = c.DefaultQuery("type", "registrant")
	}

	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	actorID, _ := gatewayauth.GetUserID(c)
	actorName := c.GetHeader("X-User-Name")
	if actorName == "" {
		actorName = "Admin PMB"
	}

	pdfBytes, fileName, err := ctrl.svc.GenerateOfficialReportPDF(c.Request.Context(), docType, periodID, actorID, actorName)
	if err != nil {
		response.InternalServerError(c, "Gagal menghasilkan berita acara resmi: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

