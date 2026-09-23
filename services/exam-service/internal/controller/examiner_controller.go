package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/service"
)

type ExaminerController struct {
	svc service.ExamService
}

func NewExaminerController(svc service.ExamService) *ExaminerController {
	return &ExaminerController{svc: svc}
}

func (ctrl *ExaminerController) GetMyAssignedSessions(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Otentikasi penguji diperlukan")
		return
	}

	sessions, err := ctrl.svc.GetExaminerSessions(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar sesi yang ditugaskan: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar sesi penguji berhasil diambil", sessions)
}

func (ctrl *ExaminerController) GetMySessionParticipants(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Otentikasi penguji diperlukan")
		return
	}

	participants, err := ctrl.svc.GetSessionParticipants(c.Request.Context(), sessionID, &userID, "dosen")
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenExaminer) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil peserta sesi: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar peserta sesi penguji berhasil diambil", participants)
}
