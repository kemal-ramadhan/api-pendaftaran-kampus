package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
)

type StudentController struct {
	svc service.SelectionService
}

func NewStudentController(svc service.SelectionService) *StudentController {
	return &StudentController{svc: svc}
}

// GetMyResult mengembalikan hasil seleksi dan tagihan daftar ulang calon mahasiswa
func (ctrl *StudentController) GetMyResult(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	res, err := ctrl.svc.GetMyResult(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrResultNotPublished) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeResultNotPublished, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrRecommendationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil pengumuman hasil seleksi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Hasil seleksi penerimaan mahasiswa baru berhasil diambil", res)
}
