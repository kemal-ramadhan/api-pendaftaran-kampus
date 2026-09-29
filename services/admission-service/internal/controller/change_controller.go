package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type ChangeController struct {
	svc service.ChangeService
}

func NewChangeController(svc service.ChangeService) *ChangeController {
	return &ChangeController{svc: svc}
}

// GetMyChanges menangani GET /applications/me/changes
func (ctrl *ChangeController) GetMyChanges(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Format ID pengguna tidak valid")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	changes, total, err := ctrl.svc.GetApplicationChanges(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data riwayat pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Riwayat pendaftaran berhasil diambil", gin.H{
		"items": changes,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetPathChangeOptions menangani GET /applications/me/path-change/options
func (ctrl *ChangeController) GetPathChangeOptions(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Format ID pengguna tidak valid")
		return
	}

	opts, err := ctrl.svc.GetPathChangeOptions(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "Data pendaftaran aktif tidak ditemukan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pilihan pindah jalur berhasil diambil", opts)
}

// RequestOrApplyPathChange menangani POST /applications/me/path-change
func (ctrl *ChangeController) RequestOrApplyPathChange(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Format ID pengguna tidak valid")
		return
	}

	var input service.PathChangeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "INVALID_PAYLOAD", "Payload permohonan pindah jalur tidak valid: "+err.Error())
		return
	}

	result, err := ctrl.svc.RequestOrApplyPathChange(c.Request.Context(), userID, &input)
	if err != nil {
		if errors.Is(err, service.ErrPaymentInProgress) {
			response.Conflict(c, "PAYMENT_IN_PROGRESS", err.Error())
			return
		}
		if errors.Is(err, service.ErrPathChangeLimitReached) {
			response.Conflict(c, "PATH_CHANGE_LIMIT_REACHED", err.Error())
			return
		}
		if errors.Is(err, service.ErrPathChangePending) {
			response.Conflict(c, "PATH_CHANGE_PENDING", err.Error())
			return
		}
		if errors.Is(err, service.ErrPathChangeNotAllowed) {
			response.Conflict(c, "PATH_CHANGE_NOT_ALLOWED", err.Error())
			return
		}
		if errors.Is(err, service.ErrReasonRequired) {
			response.BadRequest(c, "INVALID_REASON", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memproses permohonan pindah jalur: "+err.Error())
		return
	}

	statusCode := http.StatusOK
	if !result.AppliedDirectly {
		statusCode = http.StatusCreated
	}
	response.Success(c, statusCode, result.Message, result)
}

// CancelPendingPathChange menangani DELETE /applications/me/path-change/pending
func (ctrl *ChangeController) CancelPendingPathChange(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Format ID pengguna tidak valid")
		return
	}

	if err := ctrl.svc.CancelPendingPathChange(c.Request.Context(), userID); err != nil {
		response.NotFound(c, "Permohonan pending tidak ditemukan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pindah jalur berhasil dibatalkan", gin.H{"cancelled": true})
}

// GetDashboard menangani GET /applications/me/dashboard
func (ctrl *ChangeController) GetDashboard(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}
	userID, err := uuid.Parse(userIDVal.(string))
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Format ID pengguna tidak valid")
		return
	}

	dash, err := ctrl.svc.GetDashboardSummary(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat dashboard: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Dashboard pendaftaran berhasil diambil", dash)
}

// AdminListPathChangeRequests menangani GET /admin/path-change-requests
func (ctrl *ChangeController) AdminListPathChangeRequests(c *gin.Context) {
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	list, total, err := ctrl.svc.AdminListPathChangeRequests(c.Request.Context(), status, limit, offset)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar permohonan pindah jalur: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar permohonan pindah jalur berhasil diambil", gin.H{
		"items": list,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// AdminApprovePathChangeRequest menangani POST /admin/path-change-requests/:id/approve
func (ctrl *ChangeController) AdminApprovePathChangeRequest(c *gin.Context) {
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID permohonan tidak valid")
		return
	}

	reviewerIDVal, _ := c.Get("user_id")
	reviewerID, _ := uuid.Parse(reviewerIDVal.(string))

	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)

	err = ctrl.svc.AdminApprovePathChangeRequest(c.Request.Context(), reqID, reviewerID, body.Note)
	if err != nil {
		response.InternalServerError(c, "Gagal menyetujui permohonan pindah jalur: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pindah jalur berhasil disetujui", gin.H{"approved": true})
}

// AdminRejectPathChangeRequest menangani POST /admin/path-change-requests/:id/reject
func (ctrl *ChangeController) AdminRejectPathChangeRequest(c *gin.Context) {
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID permohonan tidak valid")
		return
	}

	reviewerIDVal, _ := c.Get("user_id")
	reviewerID, _ := uuid.Parse(reviewerIDVal.(string))

	var body struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Note) == "" {
		response.BadRequest(c, "NOTE_REQUIRED", "Catatan penolakan (note) wajib diisi")
		return
	}

	err = ctrl.svc.AdminRejectPathChangeRequest(c.Request.Context(), reqID, reviewerID, body.Note)
	if err != nil {
		response.InternalServerError(c, "Gagal menolak permohonan pindah jalur: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pindah jalur berhasil ditolak", gin.H{"rejected": true})
}

// AdminMoveApplication menangani POST /admin/applications/:id/move
func (ctrl *ChangeController) AdminMoveApplication(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	actorIDVal, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(actorIDVal.(string))

	var input service.AdminMoveApplicationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "INVALID_PAYLOAD", "Payload pemindahan jalur tidak valid: "+err.Error())
		return
	}

	err = ctrl.svc.AdminMoveApplication(c.Request.Context(), appID, actorID, &input)
	if err != nil {
		if errors.Is(err, service.ErrPathChangeNotAllowed) {
			response.Conflict(c, "PATH_CHANGE_NOT_ALLOWED", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memindahkan jalur pendaftaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pendaftaran berhasil dialihkan ke jalur baru oleh administrator", gin.H{"moved": true})
}

// AdminGetApplicationChanges menangani GET /admin/applications/:id/changes
func (ctrl *ChangeController) AdminGetApplicationChanges(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	changes, total, err := ctrl.svc.AdminGetApplicationChanges(c.Request.Context(), appID, limit, offset)
	if err != nil {
		response.NotFound(c, "Riwayat pendaftaran tidak ditemukan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Riwayat pendaftaran lengkap berhasil diambil", gin.H{
		"items": changes,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// AdminExportPathChanges menangani GET /admin/path-changes/export
func (ctrl *ChangeController) AdminExportPathChanges(c *gin.Context) {
	status := c.Query("status")
	data, err := ctrl.svc.AdminExportPathChangesXLSX(c.Request.Context(), status)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor permohonan pindah jalur: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=permohonan_pindah_jalur.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
