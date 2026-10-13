package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/service"
)

type ExaminerController struct {
	svc service.ScoringService
}

func NewExaminerController(svc service.ScoringService) *ExaminerController {
	return &ExaminerController{svc: svc}
}

type InputScorePayload struct {
	ComponentCode string  `json:"component_code" binding:"required"`
	Score         float64 `json:"score"`
	Note          string  `json:"note"`
}

func (ctrl *ExaminerController) InputScore(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var payload InputScorePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data nilai tidak valid: "+err.Error())
		return
	}

	actorUserID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}
	roles := gatewayauth.GetUserRoles(c)

	score, summary, err := ctrl.svc.InputScore(
		c.Request.Context(),
		appID,
		payload.ComponentCode,
		payload.Score,
		payload.Note,
		actorUserID,
		roles,
	)
	if err != nil {
		if errors.Is(err, service.ErrScoresLocked) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeScoresLocked, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrForbiddenExaminer) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrComponentNotFound) {
			response.NotFound(c, "Komponen nilai tidak ditemukan atau tidak aktif")
			return
		}
		if errors.Is(err, service.ErrInvalidScoreRange) {
			response.BadRequest(c, response.ErrCodeValidationError, err.Error())
			return
		}
		if errors.Is(err, service.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal menyimpan nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Nilai pendaftar berhasil disimpan dan nilai akhir dihitung", gin.H{
		"score":   score,
		"summary": summary,
	})
}

func (ctrl *ExaminerController) GetParticipantScores(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	scores, summary, err := ctrl.svc.GetParticipantScores(c.Request.Context(), appID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data nilai peserta: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data nilai peserta berhasil diambil", gin.H{
		"scores":  scores,
		"summary": summary,
	})
}

type InputInterviewScorePayload struct {
	Score float64 `json:"score"`
	Note  string  `json:"note"`
}

// InputInterviewScore menangani PUT /scoring/admin/interview-scores/:application_id (PRD 2.4)
func (ctrl *ExaminerController) InputInterviewScore(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	var payload InputInterviewScorePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data nilai wawancara tidak valid: "+err.Error())
		return
	}

	actorUserID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}
	roles := gatewayauth.GetUserRoles(c)

	// Coba kode komponen "interview", bila tidak ada coba "wawancara"
	score, summary, err := ctrl.svc.InputScore(
		c.Request.Context(),
		appID,
		"interview",
		payload.Score,
		payload.Note,
		actorUserID,
		roles,
	)
	if err != nil && errors.Is(err, service.ErrComponentNotFound) {
		score, summary, err = ctrl.svc.InputScore(
			c.Request.Context(),
			appID,
			"wawancara",
			payload.Score,
			payload.Note,
			actorUserID,
			roles,
		)
	}

	if err != nil {
		if errors.Is(err, service.ErrScoresLocked) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeScoresLocked, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrForbiddenExaminer) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrComponentNotFound) {
			response.NotFound(c, "Komponen nilai wawancara tidak ditemukan pada jalur ini")
			return
		}
		if errors.Is(err, service.ErrInvalidScoreRange) {
			response.BadRequest(c, response.ErrCodeValidationError, err.Error())
			return
		}
		if errors.Is(err, service.ErrApplicationNotFound) {
			response.NotFound(c, "Data pendaftaran tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal menyimpan nilai wawancara: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Nilai wawancara berhasil disimpan dan nilai akhir dihitung", gin.H{
		"score":   score,
		"summary": summary,
	})
}
