package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
)

type ApprovalController struct {
	svc service.SelectionService
}

func NewApprovalController(svc service.SelectionService) *ApprovalController {
	return &ApprovalController{svc: svc}
}

// ListApprovals mengambil daftar usulan rekomendasi yang menunggu persetujuan pimpinan
func (ctrl *ApprovalController) ListApprovals(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	var periodID uuid.UUID
	if periodIDStr != "" {
		parsed, err := uuid.Parse(periodIDStr)
		if err != nil {
			response.BadRequest(c, "INVALID_ID", "Format period_id tidak valid")
			return
		}
		periodID = parsed
	}

	list, err := ctrl.svc.ListPendingApprovals(c.Request.Context(), periodID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar usulan rekomendasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar rekomendasi menunggu persetujuan berhasil diambil", list)
}

// Approve menyetujui usulan rekomendasi kelulusan dari program studi
func (ctrl *ApprovalController) Approve(c *gin.Context) {
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

	rec, err := ctrl.svc.ApproveRecommendation(c.Request.Context(), recID, userID)
	if err != nil {
		if errors.Is(err, service.ErrRecommendationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidRecommendationState) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal menyetujui rekomendasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Rekomendasi program studi berhasil disetujui", rec)
}

// Return mengembalikan usulan rekomendasi ke program studi dengan catatan revisi
func (ctrl *ApprovalController) Return(c *gin.Context) {
	idStr := c.Param("id")
	recID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format id rekomendasi tidak valid")
		return
	}

	var payload model.ReviewRecommendationPayload
	_ = c.ShouldBindJSON(&payload) // Catatan bisa kosong atau ada teks

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	rec, err := ctrl.svc.ReturnRecommendation(c.Request.Context(), recID, payload.Note, userID)
	if err != nil {
		if errors.Is(err, service.ErrRecommendationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidRecommendationState) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengembalikan rekomendasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Rekomendasi dikembalikan ke program studi untuk revisi dan nilai telah dibuka kembali", rec)
}

// GetScoreboard mengambil papan nilai seleksi akademik, CBT, rapor & wawancara (PRD 2.4)
func (ctrl *ApprovalController) GetScoreboard(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	var periodID uuid.UUID
	if periodIDStr != "" {
		parsed, err := uuid.Parse(periodIDStr)
		if err == nil {
			periodID = parsed
		}
	}

	decisionFilter := c.Query("decision")
	search := c.Query("q")
	roles := gatewayauth.GetUserRoles(c)
	var scopes []string
	for _, sc := range gatewayauth.GetUserScopes(c) {
		scopes = append(scopes, sc.ID)
	}

	res, err := ctrl.svc.GetScoreboard(c.Request.Context(), periodID, decisionFilter, search, roles, scopes)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data scoreboard seleksi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data scoreboard seleksi berhasil diambil", res)
}
