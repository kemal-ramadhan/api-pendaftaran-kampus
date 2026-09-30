package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type ReceiptController struct {
	svc service.PaymentService
}

func NewReceiptController(svc service.PaymentService) *ReceiptController {
	return &ReceiptController{svc: svc}
}

func (ctrl *ReceiptController) GetReceipt(c *gin.Context) {
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

	receipt, pdfBytes, err := ctrl.svc.GetReceiptByPaymentID(c.Request.Context(), paymentID, userID, role)
	if err != nil {
		if errors.Is(err, repository.ErrReceiptNotFound) {
			response.NotFound(c, "Kwitansi pembayaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak diizinkan mengakses kwitansi ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil kwitansi: "+err.Error())
		return
	}

	// Jika klien meminta data JSON secara eksplisit
	if c.Query("format") == "json" {
		response.Success(c, http.StatusOK, "Kwitansi pembayaran berhasil diambil", receipt)
		return
	}

	fileName := fmt.Sprintf("Kwitansi-%s.pdf", receipt.ReceiptNumber)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
