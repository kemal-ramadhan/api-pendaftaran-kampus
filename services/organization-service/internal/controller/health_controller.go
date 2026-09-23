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

func (ctl *HealthController) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "organization-service",
	})
}

func (ctl *HealthController) ReadinessCheck(c *gin.Context) {
	if ctl.db != nil {
		sqlDB, err := ctl.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			response.Error(c, http.StatusServiceUnavailable, response.ErrCodeInternalError, "Database tidak siap", nil)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"service":  "organization-service",
		"database": "connected",
	})
}
