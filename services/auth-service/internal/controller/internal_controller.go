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
	if !response.BindJSON(c, &req) {
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
	if !response.BindJSON(c, &req) {
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

// VerifyPassword godoc
// @Summary Verifikasi Kata Sandi Pengguna (Internal)
// @Description Memverifikasi kata sandi pengguna untuk aksi sensitif (seperti pengunduran diri).
// @Tags Internal
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body dto.VerifyPasswordRequest true "Kata Sandi"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 429 {object} response.ErrorResponse
// @Router /internal/users/{id}/verify-password [post]
func (ctl *InternalController) VerifyPassword(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.VerifyPasswordRequest
	if !response.BindJSON(c, &req) {
		return
	}

	err = ctl.adminService.InternalVerifyPassword(c.Request.Context(), targetID, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrTooManyAttempts) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeRateLimited, "Akun terkunci sementara karena terlalu banyak percobaan gagal", nil)
			return
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusBadRequest, "PASSWORD_CONFIRMATION_FAILED", "Kata sandi konfirmasi salah", nil)
			return
		}
		response.InternalServerError(c, "Gagal memverifikasi kata sandi")
		return
	}

	response.Success(c, http.StatusOK, "Kata sandi terverifikasi", gin.H{"valid": true})
}

// RemoveUserRole godoc
// @Summary Cabut Role dari Pengguna (Internal)
// @Description Mencabut role pengguna dari service hilir secara idempoten.
// @Tags Internal
// @Produce json
// @Param id path string true "User UUID"
// @Param code path string true "Kode Role"
// @Success 200 {object} response.SuccessResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/users/{id}/roles/{code} [delete]
func (ctl *InternalController) RemoveUserRole(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	roleCode := c.Param("code")
	if roleCode == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Kode role wajib diisi")
		return
	}

	err = ctl.adminService.InternalRemoveRole(c.Request.Context(), targetID, roleCode)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mencabut role pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Role pengguna berhasil dicabut", nil)
}
