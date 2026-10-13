package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/service"
)

type ActivityController struct {
	svc service.ActivityService
}

func NewActivityController(svc service.ActivityService) *ActivityController {
	return &ActivityController{svc: svc}
}

// RecordActivity godoc
// @Summary Catat Jejak Aktivitas Panitia (Internal)
// @Description Menerima event aktivitas administratif dari service lain lewat outbox
// @Tags Internal
// @Accept json
// @Produce json
// @Param request body dto.RecordActivityRequest true "Data Event Aktivitas"
// @Success 201 {object} response.SuccessResponse
// @Router /internal/activity [post]
func (ctl *ActivityController) RecordActivity(c *gin.Context) {
	var req dto.RecordActivityRequest
	if !response.BindJSON(c, &req) {
		return
	}

	if err := ctl.svc.RecordActivity(c.Request.Context(), req); err != nil {
		response.InternalServerError(c, "Gagal mencatat aktivitas: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Aktivitas berhasil dicatat", nil)
}

// GetRecentActivities godoc
// @Summary Log Aktivitas Panitia (Admin)
// @Description Mengambil daftar 20 aktivitas terbaru panitia untuk komponen dashboard
// @Tags Admin
// @Produce json
// @Param period_id query string false "Filter ID Periode / Gelombang"
// @Param limit query int false "Jumlah data (maks 100)" default(20)
// @Success 200 {object} response.SuccessResponse{data=[]dto.ActivityEventResponse}
// @Router /notifications/admin/activity [get]
func (ctl *ActivityController) GetRecentActivities(c *gin.Context) {
	periodID := c.Query("period_id")
	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	roles := gatewayauth.GetUserRoles(c)
	res, err := ctl.svc.GetRecentActivities(c.Request.Context(), periodID, limit, roles)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil log aktivitas: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Berhasil mengambil log aktivitas", res)
}
