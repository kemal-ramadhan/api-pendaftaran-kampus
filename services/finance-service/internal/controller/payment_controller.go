package controller

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type PaymentController struct {
	svc service.PaymentService
}

func NewPaymentController(svc service.PaymentService) *PaymentController {
	return &PaymentController{svc: svc}
}

type PayInput struct {
	Method string `json:"method"`
}

func (ctrl *PaymentController) PayWithVirtualAccount(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	invID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tagihan tidak valid")
		return
	}

	payment, err := ctrl.svc.PayWithVirtualAccount(c.Request.Context(), invID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			response.NotFound(c, "Data tagihan tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan membayar tagihan ini")
			return
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.BadRequest(c, "ALREADY_PAID", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat pembayaran Virtual Account: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Virtual Account berhasil dibuat", payment)
}

func (ctrl *PaymentController) PayWithManualTransfer(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	invID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID tagihan tidak valid")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "FILE_REQUIRED", "Berkas bukti transfer wajib diunggah")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.InternalServerError(c, "Gagal membuka berkas bukti transfer")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		response.InternalServerError(c, "Gagal membaca isi berkas bukti transfer")
		return
	}

	payment, err := ctrl.svc.PayWithManualTransfer(c.Request.Context(), invID, userID, fileBytes, fileHeader.Filename)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			response.NotFound(c, "Data tagihan tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengakses tagihan ini")
			return
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.BadRequest(c, "ALREADY_PAID", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengunggah bukti transfer: "+err.Error())
		return
	}

	response.Success(c, http.StatusAccepted, "Bukti transfer berhasil diunggah dan menunggu verifikasi staf keuangan", payment)
}

func (ctrl *PaymentController) GetPaymentDetail(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	pIDStr := c.Param("payment_id")
	paymentID, err := uuid.Parse(pIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pembayaran tidak valid")
		return
	}

	roles := gatewayauth.GetUserRoles(c)
	role := "user"
	if len(roles) > 0 {
		role = roles[0]
	}

	payment, err := ctrl.svc.GetPaymentByID(c.Request.Context(), paymentID, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			response.NotFound(c, "Data pembayaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki akses ke data pembayaran ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail pembayaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail pembayaran berhasil diambil", payment)
}
