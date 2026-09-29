package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
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

// SearchNIK mencari identitas person berdasarkan NIK hash atau plain NIK (PRD 2.2)
func (ctrl *InternalController) SearchNIK(c *gin.Context) {
	var req dto.SearchNIKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Format data pencarian NIK tidak valid", nil)
		return
	}

	nikHash := req.NIKHash
	if nikHash == "" && req.NIK != "" {
		nikHash = crypto.HashNIK(req.NIK)
	}

	if nikHash == "" {
		response.BadRequest(c, "INVALID_QUERY", "NIK atau hash NIK wajib disediakan")
		return
	}

	detail, err := ctrl.identitySvc.SearchNIK(c.Request.Context(), nikHash)
	if err != nil {
		if errors.Is(err, repository.ErrPersonNotFound) {
			response.Success(c, http.StatusOK, "Pencarian NIK selesai", dto.SearchNIKResponse{
				Found: false,
			})
			return
		}
		response.InternalServerError(c, "Gagal memproses pencarian NIK")
		return
	}

	response.Success(c, http.StatusOK, "Data NIK ditemukan", dto.SearchNIKResponse{
		Found:  true,
		Person: detail,
	})
}
