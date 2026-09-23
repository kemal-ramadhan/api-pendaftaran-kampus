package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type ApplicationController struct {
	svc service.AdmissionService
}

func NewApplicationController(svc service.AdmissionService) *ApplicationController {
	return &ApplicationController{svc: svc}
}

type CreateApplicationInput struct {
	PeriodID        uint       `json:"period_id" binding:"required"`
	AdmissionPathID uint       `json:"admission_path_id" binding:"required"`
	Choice1ID       uuid.UUID  `json:"choice_1_id" binding:"required"`
	Choice2ID       *uuid.UUID `json:"choice_2_id"`
}

type ChangeChoicesInput struct {
	Choice1ID uuid.UUID  `json:"choice_1_id" binding:"required"`
	Choice2ID *uuid.UUID `json:"choice_2_id"`
}

func (ctrl *ApplicationController) CreateApplication(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	var input CreateApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data permohonan pendaftaran tidak lengkap: "+err.Error())
		return
	}

	email := gatewayauth.GetUserEmail(c)
	name := c.GetHeader("X-User-Name")

	req := &service.CreateApplicationRequest{
		UserID:          userID,
		PeriodID:        input.PeriodID,
		AdmissionPathID: input.AdmissionPathID,
		Choice1ID:       input.Choice1ID,
		Choice2ID:       input.Choice2ID,
		UserEmail:       email,
		UserName:        name,
	}

	app, err := ctrl.svc.CreateApplication(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrPeriodClosed) {
			response.BadRequest(c, "PERIOD_CLOSED", err.Error())
			return
		}
		if errors.Is(err, service.ErrDuplicateApplication) {
			response.Conflict(c, "DUPLICATE_APPLICATION", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat formulir pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Pendaftaran berhasil dibuat, silakan lakukan pembayaran", app)
}

func (ctrl *ApplicationController) GetMyApplications(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	apps, err := ctrl.svc.GetMyApplications(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar pendaftaran saya: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar pendaftaran berhasil diambil", apps)
}

func (ctrl *ApplicationController) GetApplicationDetail(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

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

	app, err := ctrl.svc.GetApplicationDetail(c.Request.Context(), appID, userID, role, nil)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengakses data pendaftaran ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail pendaftaran berhasil diambil", app)
}

func (ctrl *ApplicationController) ChangeChoices(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var input ChangeChoicesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data pilihan program studi tidak lengkap: "+err.Error())
		return
	}

	req := &service.ChangeChoicesRequest{
		ApplicationID: appID,
		UserID:        userID,
		Choice1ID:     input.Choice1ID,
		Choice2ID:     input.Choice2ID,
	}

	err = ctrl.svc.ChangeChoices(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengubah pendaftaran ini")
			return
		}
		if errors.Is(err, service.ErrInvalidState) {
			response.BadRequest(c, "INVALID_STATE", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengubah pilihan program studi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pilihan program studi berhasil diperbarui", nil)
}
