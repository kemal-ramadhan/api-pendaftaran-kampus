package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/service"
)

type InternalController struct {
	svc service.AdmissionService
}

func NewInternalController(svc service.AdmissionService) *InternalController {
	return &InternalController{svc: svc}
}

type UpdateStatusInput struct {
	Status    model.ApplicationStatus `json:"status" binding:"required"`
	Notes     string                  `json:"notes"`
	ChangedBy *uuid.UUID              `json:"changed_by"`
}

func (ctrl *InternalController) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var input UpdateStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data permohonan status tidak valid: "+err.Error())
		return
	}

	err = ctrl.svc.UpdateApplicationStatus(c.Request.Context(), appID, input.Status, input.Notes, input.ChangedBy)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memperbarui status pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Status pendaftaran berhasil diperbarui", gin.H{
		"application_id": appID,
		"status":         input.Status,
	})
}

func (ctrl *InternalController) PassApplication(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	err = ctrl.svc.UpdateApplicationStatus(c.Request.Context(), appID, model.StatusPassed, "Lulus seleksi pendaftaran", nil)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal meluluskan status pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Status pendaftaran berhasil diubah menjadi lulus (passed)", gin.H{
		"application_id": appID,
		"status":         model.StatusPassed,
	})
}

func (ctrl *InternalController) GetApplication(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	app, err := ctrl.svc.GetApplicationDetail(c.Request.Context(), appID, uuid.Nil, "superadmin", nil)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data pendaftaran berhasil diambil", app)
}

func (ctrl *InternalController) GetQuota(c *gin.Context) {
	progCode := c.Query("study_program_code")
	capacity := 5
	response.Success(c, http.StatusOK, "Data kuota pendaftaran berhasil diambil", gin.H{
		"study_program_code": progCode,
		"capacity":           capacity,
	})
}

func (ctrl *InternalController) GetApplicationsByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format user_id tidak valid")
		return
	}

	apps, err := ctrl.svc.GetMyApplications(c.Request.Context(), userUUID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data pendaftaran berhasil diambil", apps)
}

