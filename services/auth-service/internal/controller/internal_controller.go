package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/service"
)

type InternalController struct {
	adminService service.AdminService
}

func NewInternalController(adminService service.AdminService) *InternalController {
	return &InternalController{adminService: adminService}
}

// GetUser godoc
// @Summary Ambil Data Ringkas User (Internal)
// @Description Mengambil data user untuk kebutuhan komunikasi antar-layanan internal. Dilindungi header X-Internal-Token.
// @Tags Internal
// @Produce json
// @Param id path string true "User UUID"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/users/{id} [get]
func (ctl *InternalController) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	user, err := ctl.adminService.InternalGetUser(c.Request.Context(), targetID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data internal pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data pengguna", user)
}

// GetBatchUsers godoc
// @Summary Ambil Data Batch User (Internal)
// @Description Mengambil data banyak pengguna sekaligus berdasarkan daftar ID. Maksimal 500 ID.
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body dto.BatchUsersRequest true "Daftar ID Pengguna"
// @Success 200 {object} response.SuccessResponse{data=[]dto.UserResponseData}
// @Failure 422 {object} response.ErrorResponse
// @Router /internal/users/batch [post]
func (ctl *InternalController) GetBatchUsers(c *gin.Context) {
	var req dto.BatchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Daftar ID tidak valid", nil)
		return
	}

	uuids := make([]uuid.UUID, 0, len(req.IDs))
	for _, idStr := range req.IDs {
		u, err := uuid.Parse(idStr)
		if err == nil {
			uuids = append(uuids, u)
		}
	}

	users, err := ctl.adminService.InternalGetUsersBatch(c.Request.Context(), uuids)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data batch pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data batch pengguna", users)
}

// AddUserRole godoc
// @Summary Tambah Role ke Pengguna (Internal)
// @Description Menambahkan role baru (mis. mahasiswa) ke pengguna dari service hilir.
// @Tags Internal
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body dto.AddUserRoleRequest true "Data Role Baru"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/users/{id}/roles [post]
func (ctl *InternalController) AddUserRole(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.AddUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Data role tidak valid", nil)
		return
	}

	user, err := ctl.adminService.InternalAddRole(c.Request.Context(), targetID, req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrRoleNotFound) || errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal menambahkan role pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Role pengguna berhasil ditambahkan", user)
}
