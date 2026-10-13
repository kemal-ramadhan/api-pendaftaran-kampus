package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim/internal/service"
)

type NIMController struct {
	svc service.NIMService
}

func NewNIMController(svc service.NIMService) *NIMController {
	return &NIMController{svc: svc}
}

// -----------------------------------------------------------------------------
// Endpoint Inter-Service Internal (/internal/nim/*)
// -----------------------------------------------------------------------------

func (ctrl *NIMController) IssueNIM(c *gin.Context) {
	var req model.IssueNIMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data permohonan penerbitan NIM tidak valid: "+err.Error())
		return
	}

	res, err := ctrl.svc.IssueNIM(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParameter) {
			response.BadRequest(c, "INVALID_PARAMETER", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal menerbitkan NIM: "+err.Error())
		return
	}

	status := http.StatusCreated
	msg := "NIM berhasil diterbitkan"
	if res.IsIdempotent {
		status = http.StatusOK
		msg = "NIM sudah pernah diterbitkan sebelumnya (idempoten)"
	}

	response.Success(c, status, msg, res)
}

func (ctrl *NIMController) GetIssuanceByApplicationID(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	issuance, err := ctrl.svc.GetIssuanceByApplicationID(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, service.ErrIssuanceNotFound) {
			response.NotFound(c, "Riwayat penerbitan NIM tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data penerbitan NIM: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Riwayat penerbitan NIM berhasil diambil", issuance)
}

// -----------------------------------------------------------------------------
// Endpoint Administrasi Downstream Gateway (/admin/*)
// -----------------------------------------------------------------------------

func (ctrl *NIMController) ListIssuances(c *gin.Context) {
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

	prodiFilter := c.Query("study_program_code")

	// Isolasi role prodi: hanya boleh melihat prodinya sendiri
	isProdiOnly := false
	for _, r := range roles {
		if r == "prodi" {
			isProdiOnly = true
		}
		if r == "superadmin" || r == "admin_pmb" || r == "pimpinan" {
			isProdiOnly = false
			break
		}
	}

	if isProdiOnly {
		if len(scopes) == 0 {
			response.Forbidden(c, response.ErrCodeForbidden, "Akses ditolak: role prodi tidak memiliki scope prodi")
			return
		}
		userProdi := scopes[0]
		if prodiFilter != "" && prodiFilter != userProdi {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda hanya berhak melihat riwayat penerbitan prodi "+userProdi)
			return
		}
		prodiFilter = userProdi
	}

	filter := model.IssuanceFilter{
		StudyProgramCode: prodiFilter,
		EntryYear:        entryYear,
		Search:           c.Query("search"),
		Limit:            perPage,
		Offset:           (page - 1) * perPage,
	}

	issuances, total, err := ctrl.svc.ListIssuances(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar riwayat penerbitan NIM: "+err.Error())
		return
	}

	response.SuccessWithMeta(c, http.StatusOK, "Daftar penerbitan NIM berhasil diambil", issuances, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

func (ctrl *NIMController) ListFormats(c *gin.Context) {
	formats, err := ctrl.svc.ListFormats(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil konfigurasi format NIM: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar format NIM berhasil diambil", formats)
}

func (ctrl *NIMController) ListActiveFormats(c *gin.Context) {
	formats, err := ctrl.svc.ListActiveFormats(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil konfigurasi format NIM aktif: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar format NIM aktif berhasil diambil", formats)
}

func (ctrl *NIMController) GetFormat(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tidak valid")
		return
	}

	format, err := ctrl.svc.GetFormatByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrFormatNotFound) {
			response.NotFound(c, "Format NIM tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil format NIM: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Format NIM berhasil diambil", format)
}

func (ctrl *NIMController) CreateFormat(c *gin.Context) {
	var req model.CreateNIMFormatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data format NIM tidak valid: "+err.Error())
		return
	}

	format, err := ctrl.svc.CreateFormat(c.Request.Context(), &req)
	if err != nil {
		response.InternalServerError(c, "Gagal menambahkan format NIM baru: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Format NIM baru berhasil disimpan", format)
}

func (ctrl *NIMController) UpdateFormat(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tidak valid")
		return
	}

	var req model.UpdateNIMFormatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data perbaruan format NIM tidak valid: "+err.Error())
		return
	}

	format, err := ctrl.svc.UpdateFormat(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrFormatNotFound) {
			response.NotFound(c, "Format NIM tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memperbarui format NIM: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Format NIM berhasil diperbarui", format)
}

func (ctrl *NIMController) VoidNIM(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	issuance, err := ctrl.svc.VoidNIM(c.Request.Context(), appID)
	if err != nil {
		response.InternalServerError(c, "Gagal membatalkan (void) NIM: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "NIM berhasil dibatalkan (voided)", gin.H{
		"application_id": appID,
		"voided":         true,
		"issuance":       issuance,
	})
}

