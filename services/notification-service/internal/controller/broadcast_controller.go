package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/service"
)

type BroadcastController struct {
	svc service.BroadcastService
}

func NewBroadcastController(svc service.BroadcastService) *BroadcastController {
	return &BroadcastController{svc: svc}
}

// ListTemplates godoc
// @Summary Daftar Template Broadcast (Admin)
// @Description Mengambil seluruh template pesan broadcast siap pakai
// @Tags Admin Broadcast
// @Produce json
// @Success 200 {object} response.SuccessResponse{data=[]model.BroadcastTemplate}
// @Router /notifications/admin/broadcast/templates [get]
func (ctl *BroadcastController) ListTemplates(c *gin.Context) {
	templates, err := ctl.svc.ListTemplates(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar template broadcast")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar template", templates)
}

// CreateTemplate godoc
// @Summary Tambah Template Broadcast (Admin)
// @Description Membuat template baru dengan validasi variabel resmi PMB
// @Tags Admin Broadcast
// @Accept json
// @Produce json
// @Param request body dto.CreateBroadcastTemplateRequest true "Data Template"
// @Success 201 {object} response.SuccessResponse{data=model.BroadcastTemplate}
// @Failure 422 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast/templates [post]
func (ctl *BroadcastController) CreateTemplate(c *gin.Context) {
	var req dto.CreateBroadcastTemplateRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.svc.CreateTemplate(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUnknownVariable) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal membuat template: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Template berhasil dibuat", res)
}

// UpdateTemplate godoc
// @Summary Perbarui Template Broadcast (Admin)
// @Description Memperbarui nama, kanal, subjek, atau isi template
// @Tags Admin Broadcast
// @Accept json
// @Produce json
// @Param id path int true "Template ID"
// @Param request body dto.UpdateBroadcastTemplateRequest true "Data Template"
// @Success 200 {object} response.SuccessResponse{data=model.BroadcastTemplate}
// @Failure 404 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast/templates/{id} [put]
func (ctl *BroadcastController) UpdateTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	var req dto.UpdateBroadcastTemplateRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.svc.UpdateTemplate(c.Request.Context(), uint(id), req)
	if err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrUnknownVariable) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memperbarui template: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Template berhasil diperbarui", res)
}

// DeleteTemplate godoc
// @Summary Hapus Template Broadcast (Admin)
// @Description Menghapus template broadcast berdasarkan ID
// @Tags Admin Broadcast
// @Produce json
// @Param id path int true "Template ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast/templates/{id} [delete]
func (ctl *BroadcastController) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	if err := ctl.svc.DeleteTemplate(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, service.ErrTemplateNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal menghapus template: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Template berhasil dihapus", nil)
}

// PreviewBroadcast godoc
// @Summary Pratinjau Broadcast Massal (Admin)
// @Description Mengestimasi jumlah penerima dan menampilkan contoh pesan dengan variabel terisi
// @Tags Admin Broadcast
// @Accept json
// @Produce json
// @Param request body dto.BroadcastPreviewRequest true "Parameter Preview"
// @Success 200 {object} response.SuccessResponse{data=dto.BroadcastPreviewResponse}
// @Failure 422 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast/preview [post]
func (ctl *BroadcastController) PreviewBroadcast(c *gin.Context) {
	var req dto.BroadcastPreviewRequest
	if !response.BindJSON(c, &req) {
		return
	}

	res, err := ctl.svc.PreviewBroadcast(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUnknownVariable) {
			response.Error(c, http.StatusUnprocessableEntity, "UNKNOWN_TEMPLATE_VARIABLE", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memproses pratinjau broadcast: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pratinjau broadcast berhasil dihasilkan", res)
}

// SendBroadcast godoc
// @Summary Kirim Broadcast Massal (Admin)
// @Description Menjadwalkan atau mengirim pesan massal ke segmen pendaftar yang dipilih
// @Tags Admin Broadcast
// @Accept json
// @Produce json
// @Param request body dto.SendBroadcastRequest true "Data Broadcast"
// @Success 200 {object} response.SuccessResponse{data=model.BroadcastLog}
// @Failure 400 {object} response.ErrorResponse
// @Failure 422 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast [post]
func (ctl *BroadcastController) SendBroadcast(c *gin.Context) {
	var req dto.SendBroadcastRequest
	if !response.BindJSON(c, &req) {
		return
	}

	actorID, _ := gatewayauth.GetUserID(c)
	actorName := c.GetHeader("X-User-Name")
	if actorName == "" {
		actorName = "Admin PMB"
	}

	res, err := ctl.svc.SendBroadcast(c.Request.Context(), req, actorID, actorName)
	if err != nil {
		if errors.Is(err, service.ErrRecipientCountMismatch) {
			response.Error(c, http.StatusBadRequest, "RECIPIENT_COUNT_MISMATCH", err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrUnknownVariable) {
			response.Error(c, http.StatusUnprocessableEntity, "UNKNOWN_TEMPLATE_VARIABLE", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mengirim broadcast: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Broadcast massal berhasil diproses dan dikirim", res)
}

// ListBroadcastLogs godoc
// @Summary Riwayat Pengiriman Broadcast (Admin)
// @Description Mengambil riwayat pengiriman pesan massal dan statistik terkirim
// @Tags Admin Broadcast
// @Produce json
// @Param period_id query string false "Filter Periode / Gelombang"
// @Success 200 {object} response.SuccessResponse{data=[]model.BroadcastLog}
// @Router /notifications/admin/broadcast [get]
func (ctl *BroadcastController) ListBroadcastLogs(c *gin.Context) {
	periodID := c.Query("period_id")
	logs, err := ctl.svc.ListBroadcastLogs(c.Request.Context(), periodID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil riwayat broadcast: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil riwayat broadcast", logs)
}

// GetBroadcastLog godoc
// @Summary Detail Riwayat Broadcast (Admin)
// @Description Mengambil informasi lengkap riwayat pengiriman broadcast tertentu
// @Tags Admin Broadcast
// @Produce json
// @Param id path string true "Broadcast Log ID"
// @Success 200 {object} response.SuccessResponse{data=model.BroadcastLog}
// @Failure 404 {object} response.ErrorResponse
// @Router /notifications/admin/broadcast/{id} [get]
func (ctl *BroadcastController) GetBroadcastLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format ID tidak valid")
		return
	}

	log, err := ctl.svc.GetBroadcastLog(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil detail broadcast: "+err.Error())
		return
	}
	if log == nil {
		response.NotFound(c, "Riwayat broadcast tidak ditemukan")
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil detail broadcast", log)
}
