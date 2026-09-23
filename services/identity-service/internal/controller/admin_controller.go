package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/service"
)

type AdminController struct {
	identitySvc     service.IdentityService
	completenessSvc service.CompletenessService
}

func NewAdminController(identitySvc service.IdentityService, completenessSvc service.CompletenessService) *AdminController {
	return &AdminController{
		identitySvc:     identitySvc,
		completenessSvc: completenessSvc,
	}
}

// GetPersonDetail mengambil detail lengkap calon mahasiswa untuk admin_pmb dan superadmin
func (ctrl *AdminController) GetPersonDetail(c *gin.Context) {
	callerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Klaim identitas pemanggil tidak ditemukan")
		return
	}

	targetIDStr := c.Param("user_id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_USER_ID", "ID user tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	detail, err := ctrl.identitySvc.GetPersonDetail(c.Request.Context(), targetID, callerID, roles)
	if err != nil {
		if errors.Is(err, repository.ErrPersonNotFound) {
			response.NotFound(c, "Data identitas calon mahasiswa tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data identitas calon mahasiswa")
		return
	}

	// Tambahkan informasi kelengkapan
	comp, _ := ctrl.completenessSvc.CheckCompleteness(c.Request.Context(), targetID)
	if comp != nil {
		detail.Completeness = *comp
	}

	response.Success(c, http.StatusOK, "Detail identitas calon mahasiswa berhasil diambil", detail)
}
