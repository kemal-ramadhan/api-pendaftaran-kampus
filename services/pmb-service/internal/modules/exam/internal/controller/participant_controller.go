package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/service"
)

type ParticipantController struct {
	svc service.ExamService
}

func NewParticipantController(svc service.ExamService) *ParticipantController {
	return &ParticipantController{svc: svc}
}

type AddParticipantInput struct {
	ApplicationID uuid.UUID `json:"application_id" binding:"required"`
}

type AutoAssignInput struct {
	ApplicationIDs []uuid.UUID `json:"application_ids" binding:"required,min=1"`
}

type UpdateAttendanceInput struct {
	Attendance string `json:"attendance" binding:"required,oneof=present absent"`
}

func (ctrl *ParticipantController) AddParticipant(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	var input AddParticipantInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data peserta tidak valid: "+err.Error())
		return
	}

	participant, err := ctrl.svc.AssignParticipantManual(c.Request.Context(), sessionID, input.ApplicationID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrSessionCapacityExceeded) {
			response.Error(c, http.StatusUnprocessableEntity, "SESSION_CAPACITY_EXCEEDED", err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrApplicationNotVerified) {
			response.Error(c, http.StatusUnprocessableEntity, "APPLICATION_NOT_VERIFIED", err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrDuplicateParticipant) {
			response.Conflict(c, "DUPLICATE_PARTICIPANT", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal menambahkan peserta ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Peserta berhasil dijadwalkan dan kartu ujian diterbitkan", participant)
}

func (ctrl *ParticipantController) AutoAssignParticipants(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	var input AutoAssignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Daftar ID pendaftar tidak valid: "+err.Error())
		return
	}

	assigned, err := ctrl.svc.AutoAssignVerifiedParticipants(c.Request.Context(), sessionID, input.ApplicationIDs)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrSessionCapacityExceeded) {
			response.Error(c, http.StatusUnprocessableEntity, "SESSION_CAPACITY_EXCEEDED", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal menjadwalkan peserta otomatis: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Proses penjadwalan peserta otomatis selesai", gin.H{
		"session_id": sessionID,
		"count":      len(assigned),
		"assigned":   assigned,
	})
}

func (ctrl *ParticipantController) GetSessionParticipants(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	role := c.GetHeader("X-User-Role")
	if role == "" {
		roles := gatewayauth.GetUserRoles(c)
		if len(roles) > 0 {
			role = roles[0]
		}
	}

	var examinerUserIDPtr *uuid.UUID
	if role == "dosen" {
		userID, err := gatewayauth.GetUserID(c)
		if err == nil {
			examinerUserIDPtr = &userID
		}
	}

	participants, err := ctrl.svc.GetSessionParticipants(c.Request.Context(), sessionID, examinerUserIDPtr, role)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenExaminer) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil daftar peserta: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar peserta ujian berhasil diambil", participants)
}

func (ctrl *ParticipantController) UpdateAttendance(c *gin.Context) {
	participantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID peserta tidak valid")
		return
	}

	var input UpdateAttendanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data presensi tidak valid: "+err.Error())
		return
	}

	actorUserID, _ := gatewayauth.GetUserID(c)
	actorRole := c.GetHeader("X-User-Role")
	if actorRole == "" {
		roles := gatewayauth.GetUserRoles(c)
		if len(roles) > 0 {
			actorRole = roles[0]
		}
	}

	err = ctrl.svc.UpdateAttendance(c.Request.Context(), participantID, input.Attendance, actorUserID, actorRole)
	if err != nil {
		if errors.Is(err, service.ErrParticipantNotFound) {
			response.NotFound(c, "Peserta ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrForbiddenExaminer) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidAttendance) {
			response.BadRequest(c, "INVALID_ATTENDANCE", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui presensi peserta: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Presensi peserta berhasil diperbarui", gin.H{
		"participant_id": participantID,
		"attendance":     input.Attendance,
	})
}
