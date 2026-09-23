package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
)

type DevController struct {
	svc service.PaymentService
	cfg *config.Config
}

func NewDevController(svc service.PaymentService, cfg *config.Config) *DevController {
	return &DevController{
		svc: svc,
		cfg: cfg,
	}
}

func (ctrl *DevController) SimulatePaid(c *gin.Context) {
	// Guard ketat: hanya aktif bila APP_ENV == "development"
	if !ctrl.cfg.IsDevelopment() {
		response.Forbidden(c, response.ErrCodeForbidden, "Endpoint simulasi pembayaran hanya aktif pada lingkungan development")
		return
	}

	idStr := c.Param("id")
	paymentID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pembayaran tidak valid")
		return
	}

	payment, err := ctrl.svc.SimulatePaid(c.Request.Context(), paymentID)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			response.NotFound(c, "Data pembayaran tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrDevEndpointDisabled) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mensimulasikan pelunasan pembayaran: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Pembayaran berhasil disimulasikan lunas (Dev Mode)", payment)
}
