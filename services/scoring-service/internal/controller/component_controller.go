package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/service"
)

type ComponentController struct {
	svc service.ScoringService
}

func NewComponentController(svc service.ScoringService) *ComponentController {
	return &ComponentController{svc: svc}
}

type CreateComponentInput struct {
	PathID        uuid.UUID `json:"path_id" binding:"required"`
	Code          string    `json:"code" binding:"required"`
	Name          string    `json:"name" binding:"required"`
	WeightPercent float64   `json:"weight_percent" binding:"required"`
	MaxScore      float64   `json:"max_score"`
	IsActive      *bool     `json:"is_active"`
}

type UpdateComponentInput struct {
	Name          string  `json:"name" binding:"required"`
	WeightPercent float64 `json:"weight_percent" binding:"required"`
	MaxScore      float64 `json:"max_score"`
	IsActive      *bool   `json:"is_active"`
}

func (ctrl *ComponentController) CreateComponent(c *gin.Context) {
	var input CreateComponentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data komponen tidak valid: "+err.Error())
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	maxScore := input.MaxScore
	if maxScore <= 0 {
		maxScore = 100.00
	}

	comp := &model.ScoreComponent{
		PathID:        input.PathID,
		Code:          input.Code,
		Name:          input.Name,
		WeightPercent: input.WeightPercent,
		MaxScore:      maxScore,
		IsActive:      isActive,
	}

	created, err := ctrl.svc.CreateComponent(c.Request.Context(), comp)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateComponent) {
			response.BadRequest(c, "DUPLICATE_COMPONENT", err.Error())
			return
		}
		if errors.Is(err, service.ErrWeightExceeds100) || errors.Is(err, service.ErrInvalidWeightSum) {
			response.Error(c, http.StatusUnprocessableEntity, "INVALID_WEIGHT_SUM", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal membuat komponen nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Komponen nilai berhasil dibuat", created)
}

func (ctrl *ComponentController) GetComponents(c *gin.Context) {
	pathIDStr := c.Query("path_id")
	if pathIDStr == "" {
		response.BadRequest(c, response.ErrCodeValidationError, "Parameter path_id wajib disertakan")
		return
	}
	pathID, err := uuid.Parse(pathIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format path_id tidak valid")
		return
	}

	comps, err := ctrl.svc.GetComponentsByPathID(c.Request.Context(), pathID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar komponen nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar komponen nilai berhasil diambil", comps)
}

func (ctrl *ComponentController) UpdateComponent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID komponen tidak valid")
		return
	}

	var input UpdateComponentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, response.ErrCodeValidationError, "Data pembaruan komponen tidak valid: "+err.Error())
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	maxScore := input.MaxScore
	if maxScore <= 0 {
		maxScore = 100.00
	}

	comp := &model.ScoreComponent{
		ID:            id,
		Name:          input.Name,
		WeightPercent: input.WeightPercent,
		MaxScore:      maxScore,
		IsActive:      isActive,
	}

	updated, err := ctrl.svc.UpdateComponent(c.Request.Context(), comp)
	if err != nil {
		if errors.Is(err, service.ErrComponentNotFound) {
			response.NotFound(c, "Komponen nilai tidak ditemukan")
			return
		}
		if errors.Is(err, service.ErrWeightExceeds100) || errors.Is(err, service.ErrInvalidWeightSum) {
			response.Error(c, http.StatusUnprocessableEntity, "INVALID_WEIGHT_SUM", err.Error(), nil)
			return
		}
		response.InternalServerError(c, "Gagal memperbarui komponen nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Komponen nilai berhasil diperbarui", updated)
}

func (ctrl *ComponentController) DeleteComponent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID komponen tidak valid")
		return
	}

	err = ctrl.svc.DeleteComponent(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrComponentNotFound) {
			response.NotFound(c, "Komponen nilai tidak ditemukan")
			return
		}
		response.InternalServerError(c, "Gagal menghapus komponen nilai: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Komponen nilai berhasil dihapus", nil)
}

func (ctrl *ComponentController) ValidatePathWeight(c *gin.Context) {
	pathID, err := uuid.Parse(c.Param("path_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format path_id tidak valid")
		return
	}

	if err := ctrl.svc.ValidatePathWeightTotal(c.Request.Context(), pathID); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "INVALID_WEIGHT_SUM", err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, "Total persentase bobot komponen valid (tepat 100%)", gin.H{
		"path_id": pathID,
		"valid":   true,
	})
}
