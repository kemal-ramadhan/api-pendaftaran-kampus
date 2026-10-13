package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/service"
)

type StudentController struct {
	svc service.ExamService
}

func NewStudentController(svc service.ExamService) *StudentController {
	return &StudentController{svc: svc}
}

func (ctrl *StudentController) GetMySchedule(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Otentikasi pengguna diperlukan")
		return
	}

	participants, err := ctrl.svc.GetMyExamSchedule(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil jadwal ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Jadwal ujian berhasil diambil", participants)
}

func (ctrl *StudentController) GetMyCard(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Otentikasi pengguna diperlukan")
		return
	}

	pdfBytes, filename, err := ctrl.svc.GetMyExamCardPDF(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrParticipantNotFound) || errors.Is(err, service.ErrCardNotFound) {
			response.NotFound(c, "Kartu ujian belum diterbitkan atau Anda belum dijadwalkan")
			return
		}
		response.InternalServerError(c, "Gagal mengunduh kartu ujian: "+err.Error())
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
