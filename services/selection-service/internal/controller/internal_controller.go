package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/repository"
)

type InternalController struct {
	resultRepo repository.ResultRepository
}

func NewInternalController(resultRepo repository.ResultRepository) *InternalController {
	return &InternalController{resultRepo: resultRepo}
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
