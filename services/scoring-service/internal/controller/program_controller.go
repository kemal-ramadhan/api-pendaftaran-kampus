package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/service"
)

type ProgramController struct {
	svc service.ScoringService
}

func NewProgramController(svc service.ScoringService) *ProgramController {
	return &ProgramController{svc: svc}
}

func (ctrl *ProgramController) GetRecap(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	pathIDStr := c.Query("path_id")
	programCode := c.Query("study_program_code")

	if periodIDStr == "" || pathIDStr == "" || programCode == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Parameter period_id, path_id, dan study_program_code wajib disertakan")
		return
	}

	periodID, err := uuid.Parse(periodIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format period_id tidak valid")
		return
	}

	pathID, err := uuid.Parse(pathIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format path_id tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	var scopes []string
	for _, sc := range gatewayauth.GetUserScopes(c) {
		scopes = append(scopes, sc.ID)
	}

	recap, err := ctrl.svc.GetProgramRecap(c.Request.Context(), periodID, pathID, programCode, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProgramAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil rekapitulasi nilai prodi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Rekapitulasi nilai dan pemeringkatan program studi berhasil diambil", recap)
}

func (ctrl *ProgramController) ExportRecap(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	pathIDStr := c.Query("path_id")
	programCode := c.Query("study_program_code")

	if periodIDStr == "" || pathIDStr == "" || programCode == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Parameter period_id, path_id, dan study_program_code wajib disertakan")
		return
	}

	periodID, err := uuid.Parse(periodIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format period_id tidak valid")
		return
	}

	pathID, err := uuid.Parse(pathIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format path_id tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	var scopes []string
	for _, sc := range gatewayauth.GetUserScopes(c) {
		scopes = append(scopes, sc.ID)
	}

	fileBytes, err := ctrl.svc.ExportProgramRecapXLSX(c.Request.Context(), periodID, pathID, programCode, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProgramAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengekspor rekapitulasi nilai: "+err.Error())
		return
	}

	filename := fmt.Sprintf("rekap-nilai-%s.xlsx", programCode)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileBytes)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileBytes)
}
