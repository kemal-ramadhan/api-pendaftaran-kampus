package controller

import (
	"errors"
	"net/http"
	"strconv"

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
	role := "user"
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
