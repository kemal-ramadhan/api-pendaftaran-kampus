package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (h *HealthController) Health(c *gin.Context) {
	response.Success(c, http.StatusOK, "pmb-notification-service is healthy", gin.H{
		"service": "pmb-notification-service",
		"status":  "ok",
	})
}

func (h *HealthController) Readiness(c *gin.Context) {
	response.Success(c, http.StatusOK, "pmb-notification-service is ready", gin.H{
		"service": "pmb-notification-service",
		"status":  "ready",
	})
}
