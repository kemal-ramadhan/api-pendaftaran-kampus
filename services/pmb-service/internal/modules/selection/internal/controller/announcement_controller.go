package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection/internal/service"
)

type AnnouncementController struct {
	svc service.SelectionService
}

func NewAnnouncementController(svc service.SelectionService) *AnnouncementController {
	return &AnnouncementController{svc: svc}
}

// PublishAnnouncement mempublikasikan hasil kelulusan resmi seleksi PMB
func (ctrl *AnnouncementController) PublishAnnouncement(c *gin.Context) {
	var payload model.PublishAnnouncementPayload
	if !response.BindJSON(c, &payload) {
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna tidak ditemukan")
		return
	}

	ann, err := ctrl.svc.PublishAnnouncement(c.Request.Context(), &payload, userID)
	if err != nil {
		if errors.Is(err, service.ErrAnnouncementAlreadyPublished) {
			response.Conflict(c, "ANNOUNCEMENT_ALREADY_PUBLISHED", err.Error())
			return
		}
		if errors.Is(err, service.ErrPendingApprovalsRemaining) {
			response.Error(c, http.StatusUnprocessableEntity, response.ErrCodeInvalidState, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrUnverifiedDocumentsRemaining) {
			response.Error(c, http.StatusUnprocessableEntity, "PUBLISH_REQUIRES_VERIFIED_DOCUMENTS", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal mempublikasikan pengumuman hasil seleksi: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Pengumuman hasil seleksi berhasil dipublikasikan dan tagihan daftar ulang telah diproses", ann)
}
