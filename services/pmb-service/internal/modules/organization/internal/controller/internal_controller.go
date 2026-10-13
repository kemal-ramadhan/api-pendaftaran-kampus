package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/service"
)

type InternalController struct {
	orgService service.OrganizationService
}

func NewInternalController(orgService service.OrganizationService) *InternalController {
	return &InternalController{orgService: orgService}
}

// GetStudyProgramByCode godoc
// @Summary Ambil Data Prodi (Internal)
// @Description Mengambil data program studi untuk verifikasi inter-service (Auth, Admission). Dilindungi X-Internal-Token.
// @Tags Internal
// @Produce json
// @Param code path string true "Kode Program Studi"
// @Success 200 {object} response.SuccessResponse{data=dto.StudyProgramResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/study-programs/{code} [get]
func (ctl *InternalController) GetStudyProgramByCode(c *gin.Context) {
	code := c.Param("code")
	sp, err := ctl.orgService.GetStudyProgramByCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrStudyProgramNotFound) {
			response.NotFound(c, "Program studi tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data program studi")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data program studi", sp)
}

// GetBatchStudyPrograms godoc
// @Summary Ambil Data Batch Prodi (Internal)
// @Description Mengambil data banyak program studi sekaligus berdasarkan daftar kode
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body dto.BatchStudyProgramsRequest true "Daftar Kode Program Studi"
// @Success 200 {object} response.SuccessResponse{data=[]dto.StudyProgramResponse}
// @Failure 422 {object} response.ErrorResponse
// @Router /internal/study-programs/batch [post]
func (ctl *InternalController) GetBatchStudyPrograms(c *gin.Context) {
	var req dto.BatchStudyProgramsRequest
	if !response.BindJSON(c, &req) {
		return
	}

	programs, err := ctl.orgService.GetStudyProgramsByCodes(c.Request.Context(), req.Codes)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data batch program studi")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil data batch program studi", programs)
}

// GetFacultyByCode godoc
// @Summary Ambil Data Fakultas (Internal)
// @Description Mengambil data fakultas berdasarkan kode untuk validasi scope dekan oleh Auth service.
// @Tags Internal
// @Produce json
// @Param code path string true "Kode Fakultas"
// @Success 200 {object} response.SuccessResponse{data=dto.FacultyResponse}
// @Failure 404 {object} response.ErrorResponse
// @Router /internal/faculties/{code} [get]
func (ctl *InternalController) GetFacultyByCode(c *gin.Context) {
	code := c.Param("code")
	f, err := ctl.orgService.GetFacultyByCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrFacultyNotFound) {
			response.NotFound(c, "Fakultas tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data fakultas")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil data fakultas", f)
}

// GetInstitutionProfile godoc
// @Summary Ambil Profil Institusi (Internal)
// @Description Mengambil data profil institusi, jam layanan, dan nomor berita acara untuk service lain.
// @Tags Internal
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=dto.InstitutionProfileResponse}
// @Router /internal/institution-profile [get]
func (ctl *InternalController) GetInstitutionProfile(c *gin.Context) {
	res, err := ctl.orgService.GetInstitutionProfile(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil profil institusi")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil profil institusi", res)
}

// ListStudyPrograms godoc
// @Summary Ambil Seluruh Data Prodi (Internal)
// @Description Mengambil data program studi untuk sinkronisasi inter-service (Admission). Dilindungi X-Internal-Token.
// @Tags Internal
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]dto.StudyProgramResponse}
// @Router /internal/study-programs [get]
func (ctl *InternalController) ListStudyPrograms(c *gin.Context) {
	programs, err := ctl.orgService.ListStudyPrograms(c.Request.Context(), "", nil)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar program studi")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar program studi", programs)
}
