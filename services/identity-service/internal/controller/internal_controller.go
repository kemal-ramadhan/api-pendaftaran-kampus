package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/service"
)

type InternalController struct {
	identitySvc     service.IdentityService
	completenessSvc service.CompletenessService
}

func NewInternalController(identitySvc service.IdentityService, completenessSvc service.CompletenessService) *InternalController {
	return &InternalController{
		identitySvc:     identitySvc,
		completenessSvc: completenessSvc,
	}
}

// GetPersonByID mengambil data lengkap calon mahasiswa untuk service internal (NIK unmasked)
func (ctrl *InternalController) GetPersonByID(c *gin.Context) {
	idStr := c.Param("user_id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_USER_ID", "ID user tidak valid")
		return
	}

	detail, err := ctrl.identitySvc.GetPersonDetail(c.Request.Context(), targetID, targetID, []string{"superadmin"})
	if err != nil {
		if errors.Is(err, repository.ErrPersonNotFound) {
			response.NotFound(c, "Data identitas calon mahasiswa tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data identitas calon mahasiswa")
		return
	}

	comp, _ := ctrl.completenessSvc.CheckCompleteness(c.Request.Context(), targetID)
	if comp != nil {
		detail.Completeness = *comp
	}

	response.Success(c, http.StatusOK, "Data identitas berhasil diambil", detail)
}

// GetPersonCompleteness mengecek kelengkapan biodata untuk service internal (mis. Admission sebelum submit)
func (ctrl *InternalController) GetPersonCompleteness(c *gin.Context) {
	idStr := c.Param("user_id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_USER_ID", "ID user tidak valid")
		return
	}

	comp, err := ctrl.completenessSvc.CheckCompleteness(c.Request.Context(), targetID)
	if err != nil {
		response.InternalServerError(c, "Gagal memeriksa kelengkapan biodata")
		return
	}

	response.Success(c, http.StatusOK, "Status kelengkapan biodata berhasil diperiksa", comp)
}

// BatchGetPersons mengambil daftar profil untuk sekumpulan user ID
func (ctrl *InternalController) BatchGetPersons(c *gin.Context) {
	var req dto.BatchPersonsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format request batch tidak valid", nil)
		return
	}

	var userUUIDs []uuid.UUID
	for _, idStr := range req.UserIDs {
		u, err := uuid.Parse(idStr)
		if err == nil {
			userUUIDs = append(userUUIDs, u)
		}
	}

	results, err := ctrl.identitySvc.BatchGetPersons(c.Request.Context(), userUUIDs)
	if err != nil {
		response.InternalServerError(c, "Gagal memproses batch query identitas")
		return
	}

	response.Success(c, http.StatusOK, "Data batch identitas berhasil diambil", results)
}
