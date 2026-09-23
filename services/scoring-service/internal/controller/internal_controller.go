package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/service"
)

type InternalController struct {
	svc service.ScoringService
}

func NewInternalController(svc service.ScoringService) *InternalController {
	return &InternalController{svc: svc}
}

type SetLockPayload struct {
	PeriodID         uuid.UUID  `json:"period_id" binding:"required"`
	PathID           uuid.UUID  `json:"path_id" binding:"required"`
	StudyProgramCode string     `json:"study_program_code" binding:"required"`
	IsLocked         bool       `json:"is_locked"`
	LockedBy         *uuid.UUID `json:"locked_by"`
}

func (ctrl *InternalController) GetSummaries(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	if periodIDStr == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Parameter period_id wajib disertakan")
		return
	}

	periodID, err := uuid.Parse(periodIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format period_id tidak valid")
		return
	}

	var progCodePtr *string
	if progCode := c.Query("program"); progCode != "" {
		progCodePtr = &progCode
	}

	summaries, err := ctrl.svc.GetSummariesInternal(c.Request.Context(), periodID, progCodePtr)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil ringkasan nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Ringkasan nilai berhasil diambil", summaries)
}

func (ctrl *InternalController) SetLock(c *gin.Context) {
	var payload SetLockPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data penguncian tidak valid: "+err.Error())
		return
	}

	lock, err := ctrl.svc.SetProgramLock(
		c.Request.Context(),
		payload.PeriodID,
		payload.PathID,
		payload.StudyProgramCode,
		payload.IsLocked,
		payload.LockedBy,
	)
	if err != nil {
		response.InternalServerError(c, "Gagal mengatur status penguncian nilai: "+err.Error())
		return
	}

	msg := "Nilai program studi berhasil dikunci"
	if !payload.IsLocked {
		msg = "Penguncian nilai program studi berhasil dibuka"
	}
	response.Success(c, http.StatusOK, msg, lock)
}

func (ctrl *InternalController) GetLock(c *gin.Context) {
	periodIDStr := c.Query("period_id")
	pathIDStr := c.Query("path_id")
	programCode := c.Query("study_program_code")

	if periodIDStr == "" || pathIDStr == "" || programCode == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Parameter period_id, path_id, dan study_program_code wajib disertakan")
		return
	}

	periodID, err := uuid.Parse(periodIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format period_id tidak valid")
		return
	}

	pathID, err := uuid.Parse(pathIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format path_id tidak valid")
		return
	}

	lock, err := ctrl.svc.GetProgramLock(c.Request.Context(), periodID, pathID, programCode)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil status penguncian: "+err.Error())
		return
	}

	isLocked := false
	if lock != nil {
		isLocked = lock.IsLocked
	}

	response.Success(c, http.StatusOK, "Status penguncian berhasil diambil", gin.H{
		"period_id":          periodID,
		"path_id":            pathID,
		"study_program_code": programCode,
		"is_locked":          isLocked,
		"lock":               lock,
	})
}
