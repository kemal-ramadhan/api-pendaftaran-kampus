package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/pagination"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/_template/internal/service"
)

type TemplateController struct {
	svc service.TemplateService
}

func NewTemplateController(svc service.TemplateService) *TemplateController {
	return &TemplateController{svc: svc}
}

func (ctl *TemplateController) ListItems(c *gin.Context) {
	q := pagination.FromContext(c)
	items, total, err := ctl.svc.List(c.Request.Context(), q)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data")
		return
	}

	meta := pagination.BuildMeta(total, q)
	response.SuccessWithMeta(c, http.StatusOK, "Berhasil mengambil data", items, meta)
}

func (ctl *TemplateController) GetMyData(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	data, err := ctl.svc.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "Data tidak ditemukan")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data profil", data)
}
