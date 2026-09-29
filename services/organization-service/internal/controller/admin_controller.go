package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
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
	paramVal := c.Param("code")
	if paramVal == "" {
		paramVal = c.Param("id")
	}
	id, err := strconv.ParseUint(paramVal, 10, 32)
	if err != nil {
		sp, errFind := ctl.orgService.GetStudyProgramByCode(c.Request.Context(), paramVal)
		if errFind != nil || sp == nil {
			response.NotFound(c, "Program studi tidak ditemukan")
			return
		}
		id = uint64(sp.ID)
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
// @Description Memperbarui sebagian field program studi berdasarkan ID atau kode
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "Study Program ID atau Kode"
// @Param request body dto.PatchStudyProgramRequest true "Data Program Studi"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/study-programs/{id} [patch]
func (ctl *AdminController) PatchStudyProgram(c *gin.Context) {
	paramVal := c.Param("code")
	if paramVal == "" {
		paramVal = c.Param("id")
	}
	id, err := strconv.ParseUint(paramVal, 10, 32)
	if err != nil {
		sp, errFind := ctl.orgService.GetStudyProgramByCode(c.Request.Context(), paramVal)
		if errFind != nil || sp == nil {
			response.NotFound(c, "Program studi tidak ditemukan")
			return
		}
		id = uint64(sp.ID)
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

// PatchAdmissionOpen godoc
// @Summary Buka atau Tutup Pendaftaran Program Studi (Admin)
// @Description Mengubah status buka/tutup pendaftaran PMB untuk program studi tertentu
// @Tags Admin
// @Accept json
// @Produce json
// @Param code path string true "Kode Program Studi"
// @Param request body dto.PatchAdmissionOpenRequest true "Status Pendaftaran"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/admin/study-programs/{code}/admission [patch]
func (ctl *AdminController) PatchAdmissionOpen(c *gin.Context) {
	code := c.Param("code")
	var req dto.PatchAdmissionOpenRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.PatchAdmissionOpen(c.Request.Context(), code, req.AdmissionOpen)
	if err != nil {
		if errors.Is(err, service.ErrStudyProgramNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui status pendaftaran program studi")
		return
	}

	response.Success(c, http.StatusOK, "Status pendaftaran program studi berhasil diperbarui", res)
}

// GetInstitutionProfile godoc
// @Summary Profil Institusi (Admin)
// @Description Mengambil data profil institusi, jam layanan, dan rekening transfer bank
// @Tags Admin
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.InstitutionProfileResponse}
// @Router /organization/admin/institution-profile [get]
func (ctl *AdminController) GetInstitutionProfile(c *gin.Context) {
	res, err := ctl.orgService.GetInstitutionProfile(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil profil institusi")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil profil institusi", res)
}

// UpdateInstitutionProfile godoc
// @Summary Perbarui Profil Institusi (Admin)
// @Description Memperbarui profil universitas, pimpinan, jam layanan, dan rekening bank
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body dto.UpdateInstitutionProfileRequest true "Data Profil Institusi"
// @Success 200 {object} response.SuccessResponse{data=dto.InstitutionProfileResponse}
// @Router /organization/admin/institution-profile [put]
func (ctl *AdminController) UpdateInstitutionProfile(c *gin.Context) {
	var req dto.UpdateInstitutionProfileRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.orgService.UpdateInstitutionProfile(c.Request.Context(), req)
	if err != nil {
		response.InternalServerError(c, "Gagal memperbarui profil institusi")
		return
	}

	response.Success(c, http.StatusOK, "Profil institusi berhasil diperbarui", res)
}

// GetProgramsOverview godoc
// @Summary Ringkasan Program Studi & Kuota PMB (Admin)
// @Description Mengambil agregat master data program studi, status PMB, dan estimasi UKT
// @Tags Admin
// @Produce json
// @Param faculty query string false "Filter kode fakultas"
// @Param q query string false "Pencarian nama atau kode prodi"
// @Success 200 {object} response.SuccessResponse{data=dto.ProgramsOverviewResponse}
// @Router /organization/admin/programs-overview [get]
func (ctl *AdminController) GetProgramsOverview(c *gin.Context) {
	facultyFilter := c.Query("faculty")
	query := c.Query("q")

	scope := rbac.ScopeFilter(c)
	scopeProdi := ""
	scopeFaculty := ""
	if !scope.IsUnrestricted {
		scopeProdi = scope.ProdiCode
		scopeFaculty = scope.FacultyCode
	}

	res, err := ctl.orgService.GetProgramsOverview(c.Request.Context(), facultyFilter, query, scopeProdi, scopeFaculty)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil ringkasan program studi")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil ringkasan program studi", res)
}
