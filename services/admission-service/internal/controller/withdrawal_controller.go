package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
)

type WithdrawalController struct {
	svc service.WithdrawalService
}

func NewWithdrawalController(svc service.WithdrawalService) *WithdrawalController {
	return &WithdrawalController{svc: svc}
}

func (ctrl *WithdrawalController) getUserID(c *gin.Context) (uuid.UUID, error) {
	if val, exists := c.Get("user_id"); exists {
		if idStr, ok := val.(string); ok && idStr != "" {
			return uuid.Parse(idStr)
		}
	}
	return gatewayauth.GetUserID(c)
}

// GetEligibility menangani GET /applications/me/withdrawal/eligibility
func (ctrl *WithdrawalController) GetEligibility(c *gin.Context) {
	userID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}

	res, err := ctrl.svc.GetWithdrawalEligibility(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal memeriksa eligibilitas pengunduran diri: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Informasi eligibilitas pengunduran diri", res)
}

// RequestWithdrawal menangani POST /applications/me/withdrawal
func (ctrl *WithdrawalController) RequestWithdrawal(c *gin.Context) {
	userID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}

	var input service.WithdrawalRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Input tidak valid: "+err.Error())
		return
	}

	res, err := ctrl.svc.RequestWithdrawal(c.Request.Context(), userID, &input)
	if err != nil {
		if errors.Is(err, service.ErrWithdrawalNotAllowed) || strings.Contains(err.Error(), "WITHDRAWAL_NOT_ALLOWED") || strings.Contains(err.Error(), "tidak tersedia") || strings.Contains(err.Error(), "tidak berhak") {
			response.Conflict(c, "WITHDRAWAL_NOT_ALLOWED", err.Error())
			return
		}
		if errors.Is(err, service.ErrWithdrawalPending) {
			response.Conflict(c, "WITHDRAWAL_PENDING", err.Error())
			return
		}
		if errors.Is(err, service.ErrPasswordConfirmationFailed) {
			response.Unauthorized(c, "PASSWORD_CONFIRMATION_FAILED", err.Error())
			return
		}
		if errors.Is(err, service.ErrWithdrawalStatementRequired) {
			response.BadRequest(c, response.ErrCodeValidationError, err.Error())
			return
		}
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, res.Message, res)
}

// GetMyWithdrawal menangani GET /applications/me/withdrawal
func (ctrl *WithdrawalController) GetMyWithdrawal(c *gin.Context) {
	userID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}

	w, err := ctrl.svc.GetMyWithdrawal(c.Request.Context(), userID)
	if err != nil || w == nil {
		response.NotFound(c, "Belum ada data pengunduran diri yang diajukan")
		return
	}

	response.Success(c, http.StatusOK, "Data pengunduran diri berhasil diambil", w)
}

// CancelPendingWithdrawal menangani DELETE /applications/me/withdrawal/pending
func (ctrl *WithdrawalController) CancelPendingWithdrawal(c *gin.Context) {
	userID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}

	if err := ctrl.svc.CancelPendingWithdrawal(c.Request.Context(), userID); err != nil {
		response.BadRequest(c, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pengunduran diri yang pending berhasil dibatalkan", nil)
}

// GetWithdrawalLetterPDF menangani GET /applications/me/withdrawal/letter
func (ctrl *WithdrawalController) GetWithdrawalLetterPDF(c *gin.Context) {
	userID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas pengguna tidak ditemukan")
		return
	}

	pdfBytes, filename, err := ctrl.svc.GetWithdrawalLetterPDF(c.Request.Context(), userID)
	if err != nil {
		response.NotFound(c, "Surat bukti pengunduran diri tidak tersedia: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// VerifyWithdrawalPublic menangani GET /admission/withdrawals/verify/:code
func (ctrl *WithdrawalController) VerifyWithdrawalPublic(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Kode verifikasi wajib diisi")
		return
	}

	res, err := ctrl.svc.VerifyWithdrawalPublic(c.Request.Context(), code)
	if err != nil {
		response.NotFound(c, "Surat bukti pengunduran diri tidak valid atau tidak ditemukan")
		return
	}

	response.Success(c, http.StatusOK, "Verifikasi surat bukti pengunduran diri berhasil", res)
}

// AdminListWithdrawals menangani GET /admin/withdrawals
func (ctrl *WithdrawalController) AdminListWithdrawals(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, total, err := ctrl.svc.AdminListWithdrawals(c.Request.Context(), status, page, limit)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar pengunduran diri: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar pengunduran diri berhasil diambil", gin.H{
		"items": items,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// AdminApproveWithdrawal menangani POST /admin/withdrawals/:id/approve
func (ctrl *WithdrawalController) AdminApproveWithdrawal(c *gin.Context) {
	idStr := c.Param("id")
	wID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID pengunduran diri tidak valid")
		return
	}

	reviewerID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas admin tidak ditemukan")
		return
	}

	var req struct {
		Note  string `json:"note"`
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&req)
	note := req.Note
	if note == "" && req.Notes != "" {
		note = req.Notes
	}

	if err := ctrl.svc.AdminApproveWithdrawal(c.Request.Context(), wID, reviewerID, note); err != nil {
		response.BadRequest(c, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pengunduran diri berhasil disetujui", nil)
}

// AdminRejectWithdrawal menangani POST /admin/withdrawals/:id/reject
func (ctrl *WithdrawalController) AdminRejectWithdrawal(c *gin.Context) {
	idStr := c.Param("id")
	wID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID pengunduran diri tidak valid")
		return
	}

	reviewerID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas admin tidak ditemukan")
		return
	}

	var req struct {
		Note  string `json:"note"`
		Notes string `json:"notes"`
	}
	_ = c.ShouldBindJSON(&req)
	note := req.Note
	if note == "" && req.Notes != "" {
		note = req.Notes
	}
	if strings.TrimSpace(note) == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Catatan penolakan (note) wajib diisi")
		return
	}

	if err := ctrl.svc.AdminRejectWithdrawal(c.Request.Context(), wID, reviewerID, note); err != nil {
		response.BadRequest(c, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Permohonan pengunduran diri berhasil ditolak", nil)
}

// AdminWithdrawApplication menangani POST /admin/applications/:id/withdraw
func (ctrl *WithdrawalController) AdminWithdrawApplication(c *gin.Context) {
	idStr := c.Param("id")
	appID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID pendaftaran tidak valid")
		return
	}

	adminID, err := ctrl.getUserID(c)
	if err != nil {
		response.Unauthorized(c, "UNAUTHORIZED", "Identitas admin tidak ditemukan")
		return
	}

	var input service.AdminWithdrawInput
	var attachmentKey string

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		input.ReasonCategory = c.PostForm("reason_category")
		input.ReasonText = c.PostForm("reason_text")
		input.Reason = c.PostForm("reason")
		input.Note = c.PostForm("note")
		input.Notes = c.PostForm("notes")

		fileHeader, err := c.FormFile("attachment")
		if err == nil && fileHeader != nil {
			attachmentKey = "attachments/withdrawals/" + uuid.New().String() + "-" + fileHeader.Filename
		}
	} else {
		if err := c.ShouldBindJSON(&input); err != nil {
			response.BadRequest(c, response.ErrCodeValidationError, "Input tidak valid: "+err.Error())
			return
		}
	}

	w, err := ctrl.svc.AdminWithdrawApplication(c.Request.Context(), appID, adminID, &input, attachmentKey)
	if err != nil {
		response.BadRequest(c, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pendaftaran berhasil diundurkan oleh admin", w)
}

// AdminGetWithdrawalStats menangani GET /admin/withdrawals/stats
func (ctrl *WithdrawalController) AdminGetWithdrawalStats(c *gin.Context) {
	stats, err := ctrl.svc.AdminGetWithdrawalStats(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil statistik pengunduran diri: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Statistik pengunduran diri berhasil diambil", stats)
}

// AdminExportWithdrawalsXLSX menangani GET /admin/withdrawals/export
func (ctrl *WithdrawalController) AdminExportWithdrawalsXLSX(c *gin.Context) {
	status := c.Query("status")
	xlsxBytes, err := ctrl.svc.AdminExportWithdrawalsXLSX(c.Request.Context(), status)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data pengunduran diri: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=\"rekap-pengunduran-diri.xlsx\"")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes)
}
