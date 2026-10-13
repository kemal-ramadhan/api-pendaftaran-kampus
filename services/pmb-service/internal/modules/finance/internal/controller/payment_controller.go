package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/service"
)

type PaymentController struct {
	svc      service.PaymentService
	adminSvc service.FinanceAdminService
	cfg      *config.Config
}

func NewPaymentController(svc service.PaymentService, adminSvc service.FinanceAdminService, cfg *config.Config) *PaymentController {
	return &PaymentController{svc: svc, adminSvc: adminSvc, cfg: cfg}
}

// GetPaymentConfig mengembalikan konfigurasi payment gateway dan URL pustaka POP JS
func (ctrl *PaymentController) GetPaymentConfig(c *gin.Context) {
	popURL := "https://app-sandbox.duitku.com/lib/js/duitku.js"
	env := "sandbox"
	providerName := "mock"
	canSimulate := true

	if ctrl.cfg != nil {
		providerName = ctrl.cfg.PaymentProvider
		env = ctrl.cfg.DuitkuEnv
		if ctrl.cfg.DuitkuEnv == "production" {
			popURL = "https://app.duitku.com/lib/js/duitku.js"
		}
		if ctrl.cfg.AppEnv == "production" || ctrl.cfg.PaymentProvider == "duitku" {
			canSimulate = false
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"provider":       providerName,
		"env":            env,
		"pop_script_url": popURL,
		"can_simulate":   canSimulate,
	})
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
		if errors.Is(err, service.ErrInvoiceFree) {
			response.Conflict(c, "INVOICE_FREE", "Tagihan ini gratis (Rp 0) dan tidak memerlukan pembayaran.")
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

	fileHeader, err := c.FormFile("proof")
	if err != nil {
		fileHeader, err = c.FormFile("file")
	}
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
		if errors.Is(err, service.ErrInvoiceFree) {
			response.Conflict(c, "INVOICE_FREE", "Tagihan ini gratis (Rp 0) dan tidak memerlukan pembayaran.")
			return
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.BadRequest(c, "ALREADY_PAID", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengunggah bukti transfer: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Bukti transfer berhasil diunggah dan menunggu verifikasi staf keuangan", payment)
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

func (ctrl *PaymentController) GetPaymentStatus(c *gin.Context) {
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
		response.InternalServerError(c, "Gagal memeriksa status pembayaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Status pembayaran berhasil diambil", gin.H{
		"payment_id":   payment.ID,
		"invoice_id":   payment.InvoiceID,
		"status":       payment.Status,
		"is_paid":      payment.Status == "success",
		"paid_at":      payment.PaidAt,
		"method":       payment.Method,
		"amount":       payment.Amount,
		"provider":     payment.Provider,
		"provider_ref": payment.ProviderRef,
		"va_number":    payment.VANumber,
		"payment_url":  payment.PaymentURL,
	})
}

// GetPaymentMethods menangani GET /finance/payment-methods dan GET /payment-methods (PRD 3.5)
func (ctrl *PaymentController) GetPaymentMethods(c *gin.Context) {
	amountStr := c.Query("amount")
	amount, _ := strconv.ParseInt(amountStr, 10, 64)
	if amount <= 0 {
		amount = 250000 // default perkiraan biaya pendaftaran PMB
	}

	if ctrl.adminSvc != nil {
		grouped, err := ctrl.adminSvc.GetPaymentMethodsGrouped(c.Request.Context(), amount)
		if err == nil {
			response.Success(c, http.StatusOK, "Daftar metode pembayaran berhasil diambil", grouped)
			return
		}
	}

	methods, err := ctrl.svc.GetPaymentMethods(c.Request.Context(), amount)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "PAYMENT_PROVIDER_ERROR", "Layanan penyedia pembayaran sedang mengalami gangguan", nil)
		return
	}

	response.Success(c, http.StatusOK, "Daftar metode pembayaran berhasil diambil", methods)
}

