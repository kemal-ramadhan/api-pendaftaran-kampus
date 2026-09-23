package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/service"
)

type InternalController struct {
	svc service.ExamService
}

func NewInternalController(svc service.ExamService) *InternalController {
	return &InternalController{svc: svc}
}

func (ctrl *InternalController) GetParticipantByApplicationID(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID pendaftaran tidak valid")
		return
	}

	p, err := ctrl.svc.GetParticipantByApplicationID(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, service.ErrParticipantNotFound) {
			response.NotFound(c, "Peserta ujian tidak ditemukan untuk pendaftaran ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data peserta: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data peserta ujian berhasil diambil", p)
}

func (ctrl *InternalController) GetSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	session, err := ctrl.svc.GetSessionDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal mengambil data sesi ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Data sesi ujian berhasil diambil", session)
}
