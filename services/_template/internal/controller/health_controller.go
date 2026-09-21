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

func (h *HealthController) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

func (h *HealthController) ReadinessCheck(c *gin.Context) {
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil || sqlDB.Ping() != nil {
			response.Error(c, http.StatusServiceUnavailable, response.ErrCodeInternalError, "Database unreachable", nil)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "READY",
		"database": "CONNECTED",
	})
}
