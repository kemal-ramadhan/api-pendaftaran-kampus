package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/service"
)

type AdminController struct {
	adminService service.AdminService
}

func NewAdminController(adminService service.AdminService) *AdminController {
	return &AdminController{adminService: adminService}
}

// GetRoles godoc
// @Summary Daftar Role
// @Description Mengambil seluruh master role yang tersedia dalam sistem PMB.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]dto.RoleResponseData}
// @Router /auth/admin/roles [get]
func (ctl *AdminController) GetRoles(c *gin.Context) {
	roles, err := ctl.adminService.GetRoles(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar role")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar role", roles)
}

// ListUsers godoc
// @Summary Daftar Pengguna (Admin)
// @Description Mengambil daftar pengguna dengan filter role, status, pencarian nama/email, dan paginasi.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param role query string false "Filter role (e.g. calon_mahasiswa, keuangan, dosen)"
// @Param status query string false "Filter status (active/pending/inactive)"
// @Param q query string false "Pencarian nama atau email"
// @Param page query int false "Nomor halaman" default(1)
// @Param per_page query int false "Jumlah per halaman (maks 100)" default(20)
// @Success 200 {object} response.SuccessResponse{data=[]dto.UserResponseData}
// @Router /auth/admin/users [get]
func (ctl *AdminController) ListUsers(c *gin.Context) {
	var query dto.AdminListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.UnprocessableEntity(c, "Parameter query tidak valid", nil)
		return
	}

	users, total, err := ctl.adminService.ListUsers(c.Request.Context(), query)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar pengguna")
		return
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	perPage := query.PerPage
	if perPage < 1 {
		perPage = 20
	}

	response.SuccessWithMeta(c, http.StatusOK, "Berhasil mengambil daftar pengguna", users, &response.Meta{
		Page:    page,
		PerPage: perPage,
		Total:   total,
	})
}

// CreateUser godoc
// @Summary Buat User Internal (Admin)
// @Description Membuat akun pengguna internal (keuangan, dosen, prodi, dll). Status akun langsung aktif.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.AdminCreateUserRequest true "Data Pengguna Baru"
// @Success 201 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 409 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /auth/admin/users [post]
func (ctl *AdminController) CreateUser(c *gin.Context) {
	actorID, _ := middleware.GetUserID(c)

	var req dto.AdminCreateUserRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := ctl.adminService.CreateUser(c.Request.Context(), req, actorID)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			response.Error(c, http.StatusConflict, response.ErrCodeEmailTaken, "Email sudah digunakan oleh akun lain", nil)
			return
		}
		if errors.Is(err, service.ErrValidation) || errors.Is(err, service.ErrRoleNotFound) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrUpstreamUnavailable) {
			response.Error(c, http.StatusBadGateway, response.ErrCodeUpstreamUnavailable, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal membuat pengguna baru")
		return
	}

	response.Success(c, http.StatusCreated, "Pengguna berhasil dibuat", user)
}

// GetUser godoc
// @Summary Detail Pengguna (Admin)
// @Description Mengambil informasi lengkap seorang pengguna berdasarkan ID.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path string true "User UUID"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/admin/users/{id} [get]
func (ctl *AdminController) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	user, err := ctl.adminService.GetUser(c.Request.Context(), targetID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data pengguna", user)
}

// UpdateUser godoc
// @Summary Ubah Data Dasar Pengguna (Admin)
// @Description Mengubah nama lengkap dan nomor telepon pengguna.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body dto.AdminUpdateUserRequest true "Data Baru"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/admin/users/{id} [put]
func (ctl *AdminController) UpdateUser(c *gin.Context) {
	actorID, _ := middleware.GetUserID(c)
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.AdminUpdateUserRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := ctl.adminService.UpdateUser(c.Request.Context(), targetID, req, actorID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrValidation) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memperbarui pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Data pengguna berhasil diperbarui", user)
}

// UpdateUserRoles godoc
// @Summary Ganti Role Pengguna (Admin)
// @Description Mengganti daftar role dan scope pengguna. Otomatis mencabut seluruh sesi aktif user.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body dto.AdminUpdateRolesRequest true "Role Baru"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/admin/users/{id}/roles [put]
func (ctl *AdminController) UpdateUserRoles(c *gin.Context) {
	actorID, _ := middleware.GetUserID(c)
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.AdminUpdateRolesRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := ctl.adminService.UpdateUserRoles(c.Request.Context(), targetID, req, actorID)
	if err != nil {
		if errors.Is(err, service.ErrSelfDemoteSuperadmin) || errors.Is(err, service.ErrLastSuperadmin) {
			response.BadRequest(c, response.ErrCodeValidationError, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrValidation) || errors.Is(err, service.ErrRoleNotFound) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrUpstreamUnavailable) {
			response.Error(c, http.StatusBadGateway, response.ErrCodeUpstreamUnavailable, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memperbarui role pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Role pengguna berhasil diperbarui", user)
}

// UpdateUserStatus godoc
// @Summary Ubah Status Pengguna (Admin)
// @Description Mengubah status pengguna menjadi active atau inactive. Jika dinonaktifkan, seluruh sesi login dicabut.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User UUID"
// @Param request body dto.AdminUpdateStatusRequest true "Status Baru"
// @Success 200 {object} response.SuccessResponse{data=dto.UserResponseData}
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /auth/admin/users/{id}/status [patch]
func (ctl *AdminController) UpdateUserStatus(c *gin.Context) {
	actorID, _ := middleware.GetUserID(c)
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.AdminUpdateStatusRequest
	if !response.BindJSON(c, &req) {
		return
	}

	user, err := ctl.adminService.UpdateUserStatus(c.Request.Context(), targetID, req, actorID)
	if err != nil {
		if errors.Is(err, service.ErrSelfDeactivateSuperadmin) || errors.Is(err, service.ErrLastSuperadmin) {
			response.BadRequest(c, response.ErrCodeValidationError, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, "Pengguna tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memperbarui status pengguna")
		return
	}

	response.Success(c, http.StatusOK, "Status pengguna berhasil diperbarui", user)
}
