package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/service"
)

type AdminController struct {
	orgService service.OrganizationService
}

func NewAdminController(orgService service.OrganizationService) *AdminController {
	return &AdminController{orgService: orgService}
}

// CreateFaculty godoc
// @Summary Tambah Fakultas Baru (Admin)
// @Description Menambahkan data fakultas baru. Hanya untuk superadmin.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dto.CreateFacultyRequest true "Data Fakultas"
// @Success 201 {object} response.SuccessResponse{data=dto.FacultyResponse}
// @Failure 409 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /organization/admin/faculties [post]
func (ctl *AdminController) CreateFaculty(c *gin.Context) {
	var req dto.CreateFacultyRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.CreateFaculty(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrFacultyCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat fakultas baru")
		return
	}

	response.Success(c, http.StatusCreated, "Fakultas berhasil dibuat", res)
}

// UpdateFaculty godoc
// @Summary Ubah Data Fakultas (Admin)
// @Description Memperbarui seluruh data fakultas berdasarkan ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "Faculty ID"
// @Param request body dto.UpdateFacultyRequest true "Data Fakultas"
// @Success 200 {object} response.SuccessResponse{data=dto.FacultyResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/faculties/{id} [put]
func (ctl *AdminController) UpdateFaculty(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.UpdateFacultyRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.UpdateFaculty(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, "Fakultas tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrFacultyCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui fakultas")
		return
	}

	response.Success(c, http.StatusOK, "Fakultas berhasil diperbarui", res)
}

// PatchFaculty godoc
// @Summary Ubah Sebagian Data Fakultas (Admin)
// @Description Memperbarui sebagian field fakultas berdasarkan ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "Faculty ID"
// @Param request body dto.PatchFacultyRequest true "Data Fakultas"
// @Success 200 {object} response.SuccessResponse{data=dto.FacultyResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/faculties/{id} [patch]
func (ctl *AdminController) PatchFaculty(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.PatchFacultyRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.PatchFaculty(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, "Fakultas tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrFacultyCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui fakultas")
		return
	}

	response.Success(c, http.StatusOK, "Fakultas berhasil diperbarui", res)
}

// CreateStudyProgram godoc
// @Summary Tambah Program Studi Baru (Admin)
// @Description Menambahkan data program studi baru. Hanya untuk superadmin.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dto.CreateStudyProgramRequest true "Data Program Studi"
// @Success 201 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 409 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /organization/admin/study-programs [post]
func (ctl *AdminController) CreateStudyProgram(c *gin.Context) {
	var req dto.CreateStudyProgramRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.CreateStudyProgram(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrStudyProgramCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat program studi baru")
		return
	}

	response.Success(c, http.StatusCreated, "Program studi berhasil dibuat", res)
}

// UpdateStudyProgram godoc
// @Summary Ubah Data Program Studi (Admin)
// @Description Memperbarui seluruh data program studi berdasarkan ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "Study Program ID"
// @Param request body dto.UpdateStudyProgramRequest true "Data Program Studi"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/study-programs/{id} [put]
func (ctl *AdminController) UpdateStudyProgram(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.UpdateStudyProgramRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.UpdateStudyProgram(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, service.ErrStudyProgramNotFound) || errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrStudyProgramCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui program studi")
		return
	}

	response.Success(c, http.StatusOK, "Program studi berhasil diperbarui", res)
}

// PatchStudyProgram godoc
// @Summary Ubah Sebagian Data Program Studi (Admin)
// @Description Memperbarui sebagian field program studi berdasarkan ID
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "Study Program ID"
// @Param request body dto.PatchStudyProgramRequest true "Data Program Studi"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/study-programs/{id} [patch]
func (ctl *AdminController) PatchStudyProgram(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.PatchStudyProgramRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.PatchStudyProgram(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, service.ErrStudyProgramNotFound) || errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrStudyProgramCodeTaken) {
			response.Conflict(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui program studi")
		return
	}

	response.Success(c, http.StatusOK, "Program studi berhasil diperbarui", res)
}
