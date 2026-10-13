package controller

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/service"
)

type WebhookController struct {
	svc service.PaymentService
}

func NewWebhookController(svc service.PaymentService) *WebhookController {
	return &WebhookController{svc: svc}
}

func (ctrl *WebhookController) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		provider = "mock"
	}

	sig := c.GetHeader("X-Signature")
	if sig == "" {
		sig = c.GetHeader("X-Callback-Signature")
	}
	if sig == "" {
		sig = c.GetHeader("X-Provider-Signature")
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "READ_ERROR", "Gagal membaca body webhook")
		return
	}

	err = ctrl.svc.ProcessWebhook(c.Request.Context(), provider, sig, bodyBytes)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSignature) {
			response.Error(c, http.StatusUnauthorized, "INVALID_SIGNATURE", "Tanda tangan webhook tidak valid", nil)
			return
		}
		response.InternalServerError(c, "Gagal memproses notifikasi webhook: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Webhook berhasil diproses", gin.H{
		"provider": provider,
		"status":   "ok",
	})
}
