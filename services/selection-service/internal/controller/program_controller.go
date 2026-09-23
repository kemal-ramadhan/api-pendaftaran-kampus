package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/service"
)

type ProgramController struct {
	svc service.SelectionService
}

func NewProgramController(svc service.SelectionService) *ProgramController {
	return &ProgramController{svc: svc}
}

// GetRecommendation mengambil data usulan rekomendasi dan rekap pendaftar untuk program studi
func (ctrl *ProgramController) GetRecommendation(c *gin.Context) {
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

	rec, participants, err := ctrl.svc.GetProgramRecommendation(c.Request.Context(), periodID, pathID, programCode, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProgramAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil rekomendasi program studi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data rekomendasi program studi berhasil diambil", gin.H{
		"recommendation": rec,
		"participants":   participants,
	})
}

// SaveRecommendation menyimpan rancangan usulan kelulusan program studi
func (ctrl *ProgramController) SaveRecommendation(c *gin.Context) {
	var payload model.SaveRecommendationPayload
	if !response.BindJSON(c, &payload) {
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}
	roles := gatewayauth.GetUserRoles(c)
	var scopes []string
	for _, sc := range gatewayauth.GetUserScopes(c) {
		scopes = append(scopes, sc.ID)
	}

	rec, err := ctrl.svc.SaveProgramRecommendation(c.Request.Context(), &payload, userID, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProgramAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrQuotaExceeded) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeQuotaExceeded, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrInvalidRecommendationState) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal menyimpan rekomendasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Rancangan rekomendasi berhasil disimpan", rec)
}

// SubmitRecommendation mengajukan rekomendasi program studi ke pimpinan dan mengunci nilai di Scoring Service
func (ctrl *ProgramController) SubmitRecommendation(c *gin.Context) {
	idStr := c.Param("id")
	recID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format id rekomendasi tidak valid")
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}
	roles := gatewayauth.GetUserRoles(c)
	var scopes []string
	for _, sc := range gatewayauth.GetUserScopes(c) {
		scopes = append(scopes, sc.ID)
	}

	rec, err := ctrl.svc.SubmitProgramRecommendation(c.Request.Context(), recID, userID, roles, scopes)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenProgramAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrRecommendationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrQuotaExceeded) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeQuotaExceeded, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrInvalidRecommendationState) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengajukan rekomendasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Rekomendasi berhasil diajukan ke pimpinan dan nilai peserta telah dikunci", rec)
}
