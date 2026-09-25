package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type InvoiceController struct {
	svc service.InvoiceService
}

func NewInvoiceController(svc service.InvoiceService) *InvoiceController {
	return &InvoiceController{svc: svc}
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
