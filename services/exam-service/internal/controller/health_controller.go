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

func (ctrl *HealthController) HealthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "Exam Service is healthy", gin.H{
		"service": "pmb-exam-service",
		"status":  "UP",
	})
}

func (ctrl *HealthController) ReadinessCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "Exam Service is ready", gin.H{
		"service": "pmb-exam-service",
		"status":  "READY",
	})
}

