package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type AdminController struct {
	paymentSvc      service.PaymentService
	feeSchemeRepo   repository.FeeSchemeRepository
	refundReviewSvc service.RefundReviewService
}

func NewAdminController(
	paymentSvc service.PaymentService,
	feeSchemeRepo repository.FeeSchemeRepository,
	refundReviewSvc service.RefundReviewService,
) *AdminController {
	return &AdminController{
		paymentSvc:      paymentSvc,
		feeSchemeRepo:   feeSchemeRepo,
		refundReviewSvc: refundReviewSvc,
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



