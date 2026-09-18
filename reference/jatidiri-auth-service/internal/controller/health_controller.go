package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Redis     string `json:"redis"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
}

// HealthCheck godoc
// @Summary Health Check
// @Description Check service, database, and Redis connectivity
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	dbStatus := "ok"
	if err := config.DB.Exec("SELECT 1").Error; err != nil {
		dbStatus = "error"
	}

	redisStatus := "ok"
	if err := config.RedisDB().Ping(config.Ctx()).Err(); err != nil {
		redisStatus = "error"
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Database:  dbStatus,
		Redis:     redisStatus,
		Timestamp: time.Now().Format(time.RFC3339),
		Service:   "jatidiri-auth-service",
	})
}
