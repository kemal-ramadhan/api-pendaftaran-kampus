package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/service"
)

type PublicController struct {
	orgService service.OrganizationService
}

func NewPublicController(orgService service.OrganizationService) *PublicController {
	return &PublicController{orgService: orgService}
}

// ListFaculties godoc
// @Summary Daftar Fakultas
// @Description Mengambil daftar seluruh fakultas aktif/semua
// @Tags Public
// @Produce json
// @Param is_active query bool false "Filter status aktif"
// @Success 200 {object} response.SuccessResponse{data=[]dto.FacultyResponse}
// @Router /organization/faculties [get]
func (ctl *PublicController) ListFaculties(c *gin.Context) {
	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		val, err := strconv.ParseBool(activeStr)
		if err == nil {
			isActive = &val
		}
	}

	faculties, err := ctl.orgService.ListFaculties(c.Request.Context(), isActive)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar fakultas")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil daftar fakultas", faculties)
}

// ListStudyPrograms godoc
// @Summary Daftar Program Studi
// @Description Mengambil daftar program studi dengan filter fakultas atau status aktif
// @Tags Public
// @Produce json
// @Param faculty_code query string false "Filter kode fakultas (misal: FT, FEB)"
// @Param is_active query bool false "Filter status aktif"
// @Success 200 {object} response.SuccessResponse{data=[]dto.StudyProgramResponse}
// @Router /organization/study-programs [get]
func (ctl *PublicController) ListStudyPrograms(c *gin.Context) {
	facultyCode := c.Query("faculty_code")
	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		val, err := strconv.ParseBool(activeStr)
		if err == nil {
			isActive = &val
		}
	}

	programs, err := ctl.orgService.ListStudyPrograms(c.Request.Context(), facultyCode, isActive)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar program studi")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil daftar program studi", programs)
}

// GetStudyProgramByCode godoc
// @Summary Detail Program Studi
// @Description Mengambil detail program studi berdasarkan kode unik prodi (misal: TI)
// @Tags Public
// @Produce json
// @Param code path string true "Kode Program Studi"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /organization/study-programs/{code} [get]
func (ctl *PublicController) GetStudyProgramByCode(c *gin.Context) {
	code := c.Param("code")
	sp, err := ctl.orgService.GetStudyProgramByCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrStudyProgramNotFound) {
			response.NotFound(c, "Program studi tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail program studi")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil detail program studi", sp)
}

// GetInstitutionProfile godoc
// @Summary Profil Institusi (Publik)
// @Description Mengambil data profil universitas, jam layanan, pusat bantuan, dan rekening transfer
// @Tags Public
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.InstitutionProfileResponse}
// @Router /organization/institution-profile [get]
func (ctl *PublicController) GetInstitutionProfile(c *gin.Context) {
	res, err := ctl.orgService.GetInstitutionProfile(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil profil institusi")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil profil institusi", res)
}
