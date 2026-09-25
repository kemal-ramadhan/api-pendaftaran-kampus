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
	paymentSvc    service.PaymentService
	feeSchemeRepo repository.FeeSchemeRepository
}

func NewAdminController(paymentSvc service.PaymentService, feeSchemeRepo repository.FeeSchemeRepository) *AdminController {
	return &AdminController{
		paymentSvc:    paymentSvc,
		feeSchemeRepo: feeSchemeRepo,
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

