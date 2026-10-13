package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"gorm.io/gorm"
)

type HealthController struct {
	db *gorm.DB
}

func NewHealthController(db *gorm.DB) *HealthController {
	return &HealthController{db: db}
}

func (h *HealthController) Health(c *gin.Context) {
	response.Success(c, http.StatusOK, "pmb-admission-service is healthy", gin.H{
		"service": "pmb-admission-service",
		"status":  "ok",
	})
}

func (h *HealthController) Readiness(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		response.Error(c, http.StatusServiceUnavailable, "DB_DOWN", "Database admission tidak dapat diakses", nil)
		return
	}
	response.Success(c, http.StatusOK, "pmb-admission-service is ready", gin.H{
		"service": "pmb-admission-service",
		"status":  "ready",
	})
}
