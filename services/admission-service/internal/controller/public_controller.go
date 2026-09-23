package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type PublicController struct {
	svc service.AdmissionService
}

func NewPublicController(svc service.AdmissionService) *PublicController {
	return &PublicController{svc: svc}
}

func (ctrl *PublicController) GetActivePeriod(c *gin.Context) {
	period, err := ctrl.svc.GetActivePeriod(c.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrPeriodNotFound) {
			response.NotFound(c, "Tidak ada periode pendaftaran aktif saat ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data periode: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data periode pendaftaran aktif berhasil diambil", period)
}

func (ctrl *PublicController) GetAdmissionPaths(c *gin.Context) {
	paths, err := ctrl.svc.GetAdmissionPaths(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar jalur pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar jalur pendaftaran berhasil diambil", paths)
}
