package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/service"
)

type ReferralController struct {
	svc service.ReferralService
}

func NewReferralController(svc service.ReferralService) *ReferralController {
	return &ReferralController{svc: svc}
}

func getUserID(c *gin.Context) (uuid.UUID, bool) {
	val, ok := c.Get("user_id")
	if !ok {
		// Fallback header jika ada
		headerID := c.GetHeader("X-User-ID")
		if headerID != "" {
			uid, err := uuid.Parse(headerID)
			if err == nil {
				return uid, true
			}
		}
		return uuid.Nil, false
	}
	switch v := val.(type) {
	case uuid.UUID:
		return v, true
	case string:
		uid, err := uuid.Parse(v)
		return uid, err == nil
	}
	return uuid.Nil, false
}

// CheckCode: GET /api/v1/referral/codes/:code/check
func (ctl *ReferralController) CheckCode(c *gin.Context) {
	code := c.Param("code")
	res, err := ctl.svc.CheckCode(c.Request.Context(), code)
	if err != nil {
		response.InternalServerError(c, "Gagal memeriksa kode referal")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil memeriksa kode referal", res)
}

// GetMyReferral: GET /api/v1/referral/me
func (ctl *ReferralController) GetMyReferral(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Autentikasi diperlukan")
		return
	}
	appIDStr := c.Query("application_id")
	appID, _ := uuid.Parse(appIDStr)

	res, err := ctl.svc.GetMyReferral(c.Request.Context(), userID, appID)
	if err != nil || res == nil {
		response.NotFound(c, "Belum menggunakan kode referal")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil data referal saya", res)
}

// AttachCode: PUT /api/v1/referral/me
func (ctl *ReferralController) AttachCode(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Autentikasi diperlukan")
		return
	}

	var req dto.AttachReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.UnprocessableEntity(c, "Validasi gagal: kode dan persetujuan (consent) wajib diisi", nil)
		return
	}

	appIDStr := c.Query("application_id")
	appID, _ := uuid.Parse(appIDStr)

	res, err := ctl.svc.AttachCode(c.Request.Context(), userID, appID, req)
	if err != nil {
		if errors.Is(err, service.ErrReferralConsentReq) {
			response.UnprocessableEntity(c, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrReferralLocked) {
			response.Error(c, http.StatusConflict, "REFERRAL_LOCKED", "Kode referal sudah terverifikasi dan tidak dapat diubah lagi", nil)
			return
		}
		if errors.Is(err, service.ErrReferralNotFound) {
			response.NotFound(c, "Kode referal tidak valid atau tidak aktif")
			return
		}
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Kode referal berhasil dipasang", res)
}

// DeleteCode: DELETE /api/v1/referral/me
func (ctl *ReferralController) DeleteCode(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Autentikasi diperlukan")
		return
	}
	appIDStr := c.Query("application_id")
	appID, _ := uuid.Parse(appIDStr)

	err := ctl.svc.RemoveCode(c.Request.Context(), userID, appID)
	if err != nil {
		if errors.Is(err, service.ErrReferralLocked) {
			response.Error(c, http.StatusConflict, "REFERRAL_LOCKED", "Kode referal sudah terverifikasi dan tidak dapat dihapus", nil)
			return
		}
		response.InternalServerError(c, "Gagal menghapus kode referal")
		return
	}
	response.Success(c, http.StatusOK, "Kode referal berhasil dilepas", nil)
}

// PartnerDashboard: GET /api/v1/referral/partner/dashboard
func (ctl *ReferralController) PartnerDashboard(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Autentikasi diperlukan")
		return
	}

	dash, err := ctl.svc.GetPartnerDashboard(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal memuat dashboard guru BK")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil memuat dashboard guru BK", dash)
}

// ListPartnerCodes: GET /api/v1/referral/partner/codes
func (ctl *ReferralController) ListPartnerCodes(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Autentikasi diperlukan")
		return
	}

	list, err := ctl.svc.ListPartnerCodes(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar kode")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar kode saya", list)
}

// GetCodeDetail: GET /api/v1/referral/partner/codes/:id
func (ctl *ReferralController) GetCodeDetail(c *gin.Context) {
	userID, _ := getUserID(c)
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID tidak valid")
		return
	}

	code, summary, err := ctl.svc.GetCodeDetail(c.Request.Context(), codeID, userID, false)
	if err != nil {
		response.NotFound(c, "Kode referal tidak ditemukan atau Anda bukan penanggung jawabnya")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil detail kode referal", gin.H{
		"code":    code,
		"summary": summary,
	})
}

// ListCodeUsers: GET /api/v1/referral/partner/codes/:id/users
func (ctl *ReferralController) ListCodeUsers(c *gin.Context) {
	userID, _ := getUserID(c)
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID tidak valid")
		return
	}

	q := c.Query("q")
	apprStatus := c.Query("approve_status")
	campStatus := c.Query("campus_status")

	users, err := ctl.svc.ListCodeUsers(c.Request.Context(), codeID, userID, false, q, apprStatus, campStatus)
	if err != nil {
		response.NotFound(c, "Kode referal tidak ditemukan")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar pengguna kode", users)
}

// ExportCodeUsers: GET /api/v1/referral/partner/codes/:id/users/export
func (ctl *ReferralController) ExportCodeUsers(c *gin.Context) {
	userID, _ := getUserID(c)
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID tidak valid")
		return
	}

	users, err := ctl.svc.ListCodeUsers(c.Request.Context(), codeID, userID, false, "", "", "")
	if err != nil {
		response.NotFound(c, "Kode tidak ditemukan")
		return
	}

	// CSV Export
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=pengguna-referal.csv")

	var b strings.Builder
	b.WriteString("No,Nama,Email,NIK,NISN,Jalur,Nominal,Status Approve,Keuangan Kampus,Komisi\n")
	for _, u := range users {
		b.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s,%.2f,%s,%s,%.2f\n",
			u.No, u.Name, u.Email, u.NIK, u.NISN, u.PathName, u.Amount, u.ApproveStatus, u.CampusStatus, u.CommissionAmount))
	}
	c.String(http.StatusOK, b.String())
}

// ApproveTransactions: POST /api/v1/referral/partner/transactions/approve
func (ctl *ReferralController) ApproveTransactions(c *gin.Context) {
	actorID, _ := getUserID(c)
	var req dto.ApproveTransactionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Daftar ID transaksi wajib disertakan")
		return
	}

	if err := ctl.svc.ApproveTransactions(c.Request.Context(), req.IDs, actorID, false); err != nil {
		response.InternalServerError(c, "Gagal memverifikasi transaksi")
		return
	}
	response.Success(c, http.StatusOK, "Transaksi pembayaran berhasil diverifikasi", nil)
}

// RejectTransaction: POST /api/v1/referral/partner/transactions/:id/reject
func (ctl *ReferralController) RejectTransaction(c *gin.Context) {
	actorID, _ := getUserID(c)
	txID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Transaksi tidak valid")
		return
	}
	var req dto.RejectTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Alasan penolakan wajib diisi")
		return
	}

	if err := ctl.svc.RejectTransaction(c.Request.Context(), txID, req.Reason, actorID, false); err != nil {
		response.InternalServerError(c, "Gagal menolak transaksi")
		return
	}
	response.Success(c, http.StatusOK, "Transaksi berhasil ditolak", nil)
}

// SubmitSettlement: POST /api/v1/referral/partner/codes/:id/settlements
func (ctl *ReferralController) SubmitSettlement(c *gin.Context) {
	userID, _ := getUserID(c)
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Kode tidak valid")
		return
	}

	var req dto.SubmitSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data setoran tidak lengkap")
		return
	}

	st, err := ctl.svc.SubmitSettlement(c.Request.Context(), codeID, userID, req)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Pengajuan setoran berhasil dikirim", st)
}

// ListSettlements: GET /api/v1/referral/partner/codes/:id/settlements
func (ctl *ReferralController) ListSettlements(c *gin.Context) {
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Kode tidak valid")
		return
	}
	list, err := ctl.svc.ListSettlements(c.Request.Context(), &codeID, "")
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar setoran")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil riwayat setoran", list)
}

// ListCommissions: GET /api/v1/referral/partner/codes/:id/commissions
func (ctl *ReferralController) ListCommissions(c *gin.Context) {
	codeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Kode tidak valid")
		return
	}
	list, err := ctl.svc.ListCommissions(c.Request.Context(), &codeID, "")
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar komisi")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil riwayat komisi", list)
}

// --- Admin Handlers ---

// ListAdminCodes: GET /api/v1/referral/admin/codes
func (ctl *ReferralController) ListAdminCodes(c *gin.Context) {
	q := c.Query("q")
	status := c.Query("status")
	list, err := ctl.svc.ListAdminCodes(c.Request.Context(), q, status)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar kode referal")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil daftar kode referal admin", list)
}

// CreateCode: POST /api/v1/referral/admin/codes
func (ctl *ReferralController) CreateCode(c *gin.Context) {
	actorID, _ := getUserID(c)
	var code model.ReferralCode
	if err := c.ShouldBindJSON(&code); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}
	code.CreatedBy = &actorID
	if err := ctl.svc.CreateCode(c.Request.Context(), &code); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Kode referal berhasil dibuat", code)
}

// ImportPICs: POST /api/v1/referral/admin/pics/import
func (ctl *ReferralController) ImportPICs(c *gin.Context) {
	actorID, _ := getUserID(c)
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "File CSV wajib diunggah")
		return
	}
	defer file.Close()

	created, updated, failed, errs, err := ctl.svc.ImportPICsCSV(c.Request.Context(), file, actorID)
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Format file CSV tidak valid: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Impor guru BK selesai", gin.H{
		"created": created,
		"updated": updated,
		"failed":  failed,
		"errors":  errs,
	})
}

// GetSettings: GET /api/v1/referral/admin/settings
func (ctl *ReferralController) GetSettings(c *gin.Context) {
	st, err := ctl.svc.GetSettings(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil pengaturan referal")
		return
	}
	response.Success(c, http.StatusOK, "Berhasil mengambil pengaturan referal", st)
}

// UpdateSettings: PUT /api/v1/referral/admin/settings
func (ctl *ReferralController) UpdateSettings(c *gin.Context) {
	var st model.ReferralSetting
	if err := c.ShouldBindJSON(&st); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, err.Error())
		return
	}
	if err := ctl.svc.UpdateSettings(c.Request.Context(), &st); err != nil {
		response.InternalServerError(c, "Gagal menyimpan pengaturan referal")
		return
	}
	response.Success(c, http.StatusOK, "Pengaturan referal berhasil diperbarui", st)
}

// VerifySettlement: POST /api/v1/referral/admin/settlements/:id/verify
func (ctl *ReferralController) VerifySettlement(c *gin.Context) {
	actorID, _ := getUserID(c)
	stID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Setoran tidak valid")
		return
	}
	if err := ctl.svc.VerifySettlement(c.Request.Context(), stID, actorID); err != nil {
		response.InternalServerError(c, "Gagal memverifikasi setoran")
		return
	}
	response.Success(c, http.StatusOK, "Setoran guru BK berhasil diverifikasi", nil)
}

// RejectSettlement: POST /api/v1/referral/admin/settlements/:id/reject
func (ctl *ReferralController) RejectSettlement(c *gin.Context) {
	actorID, _ := getUserID(c)
	stID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "ID Setoran tidak valid")
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Alasan penolakan wajib diisi")
		return
	}
	if err := ctl.svc.RejectSettlement(c.Request.Context(), stID, req.Reason, actorID); err != nil {
		response.InternalServerError(c, "Gagal menolak setoran")
		return
	}
	response.Success(c, http.StatusOK, "Setoran guru BK berhasil ditolak", nil)
}
