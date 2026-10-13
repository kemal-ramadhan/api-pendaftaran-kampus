package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
)

type InternalController struct {
	resultRepo repository.ResultRepository
	svc        service.SelectionService
}

func NewInternalController(resultRepo repository.ResultRepository, svc service.SelectionService) *InternalController {
	return &InternalController{
		resultRepo: resultRepo,
		svc:        svc,
	}
}

// GetResultByApplicationID mengembalikan hasil seleksi resmi berdasarkan application_id
func (ctrl *InternalController) GetResultByApplicationID(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	result, err := ctrl.resultRepo.FindByApplicationID(c.Request.Context(), appID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data hasil seleksi: "+err.Error())
		return
	}

	if result == nil {
		response.NotFound(c, "Data hasil seleksi tidak ditemukan untuk pendaftar ini")
		return
	}

	response.Success(c, http.StatusOK, "Hasil seleksi berhasil diambil", result)
}

func (ctrl *InternalController) HandleApplicationWithdrawal(c *gin.Context) {
	appIDStr := c.Param("application_id")
	appID, err := uuid.Parse(appIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format application_id tidak valid")
		return
	}

	if ctrl.svc != nil {
		if err := ctrl.svc.WithdrawApplication(c.Request.Context(), appID); err != nil {
			response.InternalServerError(c, "Gagal memproses pengunduran diri pada seleksi: "+err.Error())
			return
		}
	} else if ctrl.resultRepo != nil {
		if _, err := ctrl.resultRepo.WithdrawApplication(c.Request.Context(), appID); err != nil {
			response.InternalServerError(c, "Gagal memproses pengunduran diri pada seleksi: "+err.Error())
			return
		}
	}

	response.Success(c, http.StatusOK, "Pengunduran diri pendaftar pada seleksi berhasil diproses", gin.H{"processed": true})
}

