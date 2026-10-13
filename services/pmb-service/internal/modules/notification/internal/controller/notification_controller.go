package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/service"
)

type NotificationController struct {
	svc service.NotificationService
}

func NewNotificationController(svc service.NotificationService) *NotificationController {
	return &NotificationController{svc: svc}
}

type QueueNotificationInput struct {
	IdempotencyKey string                 `json:"idempotency_key"`
	UserID         *uuid.UUID             `json:"user_id"`
	Email          string                 `json:"email"`
	Phone          string                 `json:"phone"`
	Channel        string                 `json:"channel" binding:"required"`
	Template       string                 `json:"template" binding:"required"`
	Data           map[string]interface{} `json:"data"`
}

func (ctrl *NotificationController) SendNotification(c *gin.Context) {
	var input QueueNotificationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "INVALID_PAYLOAD", "Payload permintaan tidak valid: "+err.Error())
		return
	}

	req := &service.QueueNotificationRequest{
		IdempotencyKey: input.IdempotencyKey,
		UserID:         input.UserID,
		Email:          input.Email,
		Phone:          input.Phone,
		Channel:        input.Channel,
		Template:       input.Template,
		Data:           input.Data,
	}

	res, err := ctrl.svc.QueueNotification(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidChannel) || errors.Is(err, service.ErrMissingTarget) {
			response.BadRequest(c, "VALIDATION_ERROR", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memproses antrean notifikasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusAccepted, "Notifikasi berhasil dimasukkan ke antrean", res)
}

func (ctrl *NotificationController) GetStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID notifikasi tidak valid")
		return
	}

	res, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			response.NotFound(c, "Log notifikasi tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil log notifikasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail notifikasi berhasil diambil", res)
}
