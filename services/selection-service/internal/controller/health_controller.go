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

func (h *HealthController) HealthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "selection-service is healthy", gin.H{
		"service": "selection-service",
		"status":  "healthy",
	})
}
