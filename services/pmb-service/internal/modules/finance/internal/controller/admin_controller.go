package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/service"
)

type AdminController struct {
	paymentSvc      service.PaymentService
	feeSchemeRepo   repository.FeeSchemeRepository
	refundReviewSvc service.RefundReviewService
	adminSvc        service.FinanceAdminService
}

func NewAdminController(
	paymentSvc service.PaymentService,
	feeSchemeRepo repository.FeeSchemeRepository,
	refundReviewSvc service.RefundReviewService,
	adminSvc service.FinanceAdminService,
) *AdminController {
	return &AdminController{
		paymentSvc:      paymentSvc,
		feeSchemeRepo:   feeSchemeRepo,
		refundReviewSvc: refundReviewSvc,
		adminSvc:        adminSvc,
	}
}

type VerifyPaymentInput struct {
	Action string `json:"action" binding:"required"` // approve, reject
	Notes  string `json:"notes"`
}

func (ctrl *AdminController) ListManualTransfers(c *gin.Context) {
	limit := 10
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	page := 1
	if pStr := c.Query("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	offset := (page - 1) * limit

	list, total, err := ctrl.paymentSvc.ListAwaitingVerification(c.Request.Context(), limit, offset)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar transfer manual: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar transfer manual yang menunggu verifikasi berhasil diambil", gin.H{
		"items": list,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (ctrl *AdminController) VerifyPayment(c *gin.Context) {
	adminID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	paymentID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pembayaran tidak valid")
		return
	}

	var input VerifyPaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data verifikasi tidak lengkap: "+err.Error())
		return
	}

	if input.Action != "approve" && input.Action != "reject" {
		response.BadRequest(c, "INVALID_ACTION", "Aksi hanya boleh 'approve' atau 'reject'")
		return
	}

	err = ctrl.paymentSvc.VerifyManualTransfer(c.Request.Context(), paymentID, input.Action, input.Notes, adminID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			response.NotFound(c, "Data pembayaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrPaymentAlreadyPaid) {
			response.BadRequest(c, "ALREADY_PAID", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memproses verifikasi pembayaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Verifikasi pembayaran berhasil diproses", gin.H{
		"payment_id": paymentID,
		"action":     input.Action,
	})
}

func (ctrl *AdminController) ListFeeSchemes(c *gin.Context) {
	schemes, err := ctrl.feeSchemeRepo.FindAll(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar skema tarif: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar skema biaya berhasil diambil", schemes)
}

func (ctrl *AdminController) CreateFeeScheme(c *gin.Context) {
	var input model.FeeScheme
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data skema biaya tidak valid: "+err.Error())
		return
	}
	if err := ctrl.feeSchemeRepo.Create(c.Request.Context(), &input); err != nil {
		response.InternalServerError(c, "Gagal membuat skema biaya: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Skema biaya berhasil dibuat", input)
}

func (ctrl *AdminController) ReconcilePayments(c *gin.Context) {
	reconciledCount, err := ctrl.paymentSvc.ReconcilePendingPayments(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Gagal melakukan rekonsiliasi pembayaran: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Rekonsiliasi pembayaran selesai diproses", gin.H{
		"reconciled_count": reconciledCount,
	})
}

type UpdateRefundReviewInput struct {
	Status string `json:"status" binding:"required"` // completed, rejected
	Notes  string `json:"notes"`
}

func (ctrl *AdminController) ListRefundReviews(c *gin.Context) {
	if ctrl.refundReviewSvc == nil {
		response.Success(c, http.StatusOK, "Daftar tinjauan pengembalian dana kosong", gin.H{"items": []interface{}{}, "meta": gin.H{"total": 0, "page": 1, "limit": 10}})
		return
	}

	limit := 10
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	page := 1
	if pStr := c.Query("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	offset := (page - 1) * limit
	status := c.Query("status")

	items, total, err := ctrl.refundReviewSvc.ListRefundReviews(c.Request.Context(), status, limit, offset)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar tinjauan pengembalian dana: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar tinjauan pengembalian dana berhasil diambil", gin.H{
		"items": items,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (ctrl *AdminController) UpdateRefundReview(c *gin.Context) {
	if ctrl.refundReviewSvc == nil {
		response.InternalServerError(c, "Layanan tinjauan pengembalian dana tidak tersedia")
		return
	}

	adminID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	revID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tinjauan tidak valid")
		return
	}

	var input UpdateRefundReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data perbaruan tinjauan tidak valid: "+err.Error())
		return
	}

	updated, err := ctrl.refundReviewSvc.UpdateRefundReview(c.Request.Context(), revID, input.Status, input.Notes, adminID)
	if err != nil {
		if errors.Is(err, repository.ErrRefundReviewNotFound) {
			response.NotFound(c, "Tinjauan pengembalian dana tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memperbarui tinjauan pengembalian dana: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Tinjauan pengembalian dana berhasil diperbarui", updated)
}

// GetReconciliation menangani GET /finance/admin/reconciliation (PRD 2.3)
func (ctrl *AdminController) GetReconciliation(c *gin.Context) {
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	status := c.Query("status")
	channel := c.Query("channel")
	feeType := c.Query("type")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	res, err := ctrl.adminSvc.GetReconciliation(c.Request.Context(), periodID, status, channel, feeType, search, page, limit)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data rekonsiliasi keuangan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data rekonsiliasi keuangan berhasil diambil", res)
}

// ExportReconciliation menangani GET /finance/admin/reconciliation/export (PRD 2.3)
func (ctrl *AdminController) ExportReconciliation(c *gin.Context) {
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	format := c.DefaultQuery("format", "xlsx")
	data, fileName, err := ctrl.adminSvc.ExportReconciliation(c.Request.Context(), periodID, format)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data rekonsiliasi: "+err.Error())
		return
	}

	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if strings.ToLower(format) == "csv" {
		contentType = "text/csv"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, contentType, data)
}

// ManualConfirmPayment menangani POST /finance/admin/invoices/:id/manual-confirm (PRD 2.3)
func (ctrl *AdminController) ManualConfirmPayment(c *gin.Context) {
	actorID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	actorName := c.GetHeader("X-User-Name")
	if actorName == "" {
		actorName = "Staf Keuangan"
	}

	idStr := c.Param("id")
	invID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tagihan tidak valid")
		return
	}

	var req service.ManualConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data konfirmasi lunas tidak valid: "+err.Error())
		return
	}

	payment, err := ctrl.adminSvc.ManualConfirmPayment(c.Request.Context(), invID, actorID, actorName, req)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.Error(c, http.StatusConflict, "INVOICE_ALREADY_PAID", err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrDuplicateBankReference) {
			response.BadRequest(c, "DUPLICATE_BANK_REFERENCE", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidAmount) {
			response.BadRequest(c, "INVALID_AMOUNT", err.Error())
			return
		}
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			response.NotFound(c, "Data tagihan tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal memproses konfirmasi lunas manual: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Konfirmasi pembayaran manual berhasil diselesaikan", payment)
}

// GetLedger menangani GET /finance/admin/ledger (PRD 2.7)
func (ctrl *AdminController) GetLedger(c *gin.Context) {
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	channel := c.Query("channel")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var fromDate, toDate *time.Time
	if fStr := c.Query("from"); fStr != "" {
		if t, err := time.Parse("2006-01-02", fStr); err == nil {
			fromDate = &t
		}
	}
	if tStr := c.Query("to"); tStr != "" {
		if t, err := time.Parse("2006-01-02", tStr); err == nil {
			toDate = &t
		}
	}

	res, err := ctrl.adminSvc.GetLedger(c.Request.Context(), periodID, channel, fromDate, toDate, page, limit)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil data buku besar: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data buku besar kas berhasil diambil", res)
}

// ExportLedger menangani GET /finance/admin/ledger/export (PRD 2.7)
func (ctrl *AdminController) ExportLedger(c *gin.Context) {
	var periodID *uint
	if pIDStr := c.Query("period_id"); pIDStr != "" {
		if pID, err := strconv.ParseUint(pIDStr, 10, 32); err == nil {
			u := uint(pID)
			periodID = &u
		}
	}

	channel := c.Query("channel")
	format := c.DefaultQuery("format", "xlsx")

	var fromDate, toDate *time.Time
	if fStr := c.Query("from"); fStr != "" {
		if t, err := time.Parse("2006-01-02", fStr); err == nil {
			fromDate = &t
		}
	}
	if tStr := c.Query("to"); tStr != "" {
		if t, err := time.Parse("2006-01-02", tStr); err == nil {
			toDate = &t
		}
	}

	data, fileName, err := ctrl.adminSvc.ExportLedger(c.Request.Context(), periodID, channel, fromDate, toDate, format)
	if err != nil {
		response.InternalServerError(c, "Gagal mengekspor data buku besar: "+err.Error())
		return
	}

	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if strings.ToLower(format) == "csv" {
		contentType = "text/csv"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, contentType, data)
}



