package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/service"
)

type InvoiceController struct {
	svc      service.InvoiceService
	adminSvc service.FinanceAdminService
}

func NewInvoiceController(svc service.InvoiceService, adminSvc service.FinanceAdminService) *InvoiceController {
	return &InvoiceController{svc: svc, adminSvc: adminSvc}
}

func (ctrl *InvoiceController) GetMyInvoices(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	invoices, err := ctrl.svc.GetMyInvoices(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar tagihan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar tagihan berhasil diambil", invoices)
}

func (ctrl *InvoiceController) GetInvoiceDetail(c *gin.Context) {
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

	roles := gatewayauth.GetUserRoles(c)
	role := "user"
	if len(roles) > 0 {
		role = roles[0]
	}

	inv, err := ctrl.svc.GetInvoiceDetailDTO(c.Request.Context(), invID, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			response.NotFound(c, "Data tagihan tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengakses data tagihan ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail tagihan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail tagihan berhasil diambil", inv)
}

// RenewInvoice menangani POST /invoices/me/:id/renew (PRD 3.5 & K15)
func (ctrl *InvoiceController) RenewInvoice(c *gin.Context) {
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

	inv, err := ctrl.adminSvc.RenewInvoiceDueDate(c.Request.Context(), invID, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvoiceFree) {
			response.Conflict(c, "INVOICE_FREE", "Tagihan ini gratis (Rp 0) dan tidak memerlukan pembayaran.")
			return
		}
		if errors.Is(err, service.ErrInvoiceAlreadyPaid) {
			response.BadRequest(c, "ALREADY_PAID", err.Error())
			return
		}
		if errors.Is(err, service.ErrRenewalLimitExceeded) {
			response.BadRequest(c, "RENEWAL_LIMIT_EXCEEDED", err.Error())
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperpanjang batas jatuh tempo: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Batas waktu tagihan berhasil diperpanjang 3 hari kerja", inv)
}

// GetReceiptView menangani GET /invoices/me/:id/receipt-view dan /finance/me/invoices/:id/receipt-view (PRD 3.8)
func (ctrl *InvoiceController) GetReceiptView(c *gin.Context) {
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

	view, err := ctrl.adminSvc.GetReceiptView(c.Request.Context(), invID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrInvoiceNotFound) {
			response.NotFound(c, "Data tagihan tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data kuitansi pelunasan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Kuitansi pelunasan resmi berhasil diambil", view)
}
