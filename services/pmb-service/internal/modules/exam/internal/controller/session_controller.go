package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam/internal/service"
)

type SessionController struct {
	svc service.ExamService
}

func NewSessionController(svc service.ExamService) *SessionController {
	return &SessionController{svc: svc}
}

type CreateSessionInput struct {
	PeriodID         uuid.UUID         `json:"period_id" binding:"required"`
	PathID           uuid.UUID         `json:"path_id" binding:"required"`
	Type             model.SessionType `json:"type" binding:"required,oneof=written_test interview"`
	StudyProgramCode *string           `json:"study_program_code"`
	Title            string            `json:"title" binding:"required"`
	StartAt          time.Time         `json:"start_at" binding:"required"`
	EndAt            time.Time         `json:"end_at" binding:"required"`
	Location         *string           `json:"location"`
	MeetingLink      *string           `json:"meeting_link"`
	Capacity         int               `json:"capacity" binding:"required,min=1"`
}

func (ctrl *SessionController) CreateSession(c *gin.Context) {
	var input CreateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data sesi ujian tidak valid: "+err.Error())
		return
	}

	session := &model.ExamSession{
		PeriodID:         input.PeriodID,
		PathID:           input.PathID,
		Type:             input.Type,
		StudyProgramCode: input.StudyProgramCode,
		Title:            input.Title,
		StartAt:          input.StartAt,
		EndAt:            input.EndAt,
		Location:         input.Location,
		MeetingLink:      input.MeetingLink,
		Capacity:         input.Capacity,
		IsActive:         true,
	}

	res, err := ctrl.svc.CreateSession(c.Request.Context(), session)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSessionTime) {
			response.BadRequest(c, "INVALID_TIME", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal membuat sesi ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Sesi ujian berhasil dibuat", res)
}

func (ctrl *SessionController) ListSessions(c *gin.Context) {
	var periodIDPtr, pathIDPtr *uuid.UUID
	if pStr := c.Query("period_id"); pStr != "" {
		if id, err := uuid.Parse(pStr); err == nil {
			periodIDPtr = &id
		}
	}
	if pStr := c.Query("path_id"); pStr != "" {
		if id, err := uuid.Parse(pStr); err == nil {
			pathIDPtr = &id
		}
	}

	var typePtr *string
	if tStr := c.Query("type"); tStr != "" {
		typePtr = &tStr
	}

	var isActivePtr *bool
	if aStr := c.Query("is_active"); aStr != "" {
		if b, err := strconv.ParseBool(aStr); err == nil {
			isActivePtr = &b
		}
	}

	sessions, err := ctrl.svc.ListSessions(c.Request.Context(), periodIDPtr, pathIDPtr, typePtr, isActivePtr)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar sesi ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar sesi ujian berhasil diambil", sessions)
}

func (ctrl *SessionController) GetSession(c *gin.Context) {
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

func (ctrl *SessionController) UpdateSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	var input CreateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data sesi ujian tidak valid: "+err.Error())
		return
	}

	session := &model.ExamSession{
		ID:               id,
		Type:             input.Type,
		StudyProgramCode: input.StudyProgramCode,
		Title:            input.Title,
		StartAt:          input.StartAt,
		EndAt:            input.EndAt,
		Location:         input.Location,
		MeetingLink:      input.MeetingLink,
		Capacity:         input.Capacity,
		IsActive:         true,
	}

	res, err := ctrl.svc.UpdateSession(c.Request.Context(), session)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrInvalidSessionTime) {
			response.BadRequest(c, "INVALID_TIME", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memperbarui sesi ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Sesi ujian berhasil diperbarui", res)
}

func (ctrl *SessionController) DeleteSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	if err := ctrl.svc.DeleteSession(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		response.BadRequest(c, "DELETE_FAILED", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Sesi ujian berhasil dihapus", nil)
}

type AssignExaminerInput struct {
	ExaminerUserID uuid.UUID `json:"examiner_user_id" binding:"required"`
}

func (ctrl *SessionController) AssignExaminer(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	var input AssignExaminerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Data penguji tidak valid: "+err.Error())
		return
	}

	examiner, err := ctrl.svc.AssignExaminer(c.Request.Context(), sessionID, input.ExaminerUserID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			response.NotFound(c, "Sesi ujian tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal menugaskan dosen penguji: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Dosen penguji berhasil ditugaskan ke sesi ujian", examiner)
}

func (ctrl *SessionController) RemoveExaminer(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID sesi tidak valid")
		return
	}

	examinerUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_USER_ID", "Format ID dosen tidak valid")
		return
	}

	if err := ctrl.svc.RemoveExaminer(c.Request.Context(), sessionID, examinerUserID); err != nil {
		response.InternalServerError(c, "Gagal menghapus dosen penguji: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Dosen penguji berhasil dihapus dari sesi ujian", nil)
}
