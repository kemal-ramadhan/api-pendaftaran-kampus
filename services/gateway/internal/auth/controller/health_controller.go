package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthController struct {
	db  *gorm.DB
	rdb *redis.Client
	env string
}

func NewHealthController(db *gorm.DB, rdb *redis.Client, env string) *HealthController {
	return &HealthController{
		db:  db,
		rdb: rdb,
		env: env,
	}
}

// LivenessCheck godoc
// @Summary Pemeriksaan Kehidupan Service (Liveness Probe)
// @Description Memeriksa apakah HTTP server auth-service aktif dan siap merespons
// @Tags Health
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Router /health [get]
func (ctl *HealthController) LivenessCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "Auth service aktif", gin.H{
		"service": "auth-service",
		"status":  "alive",
		"env":     ctl.env,
		"time":    time.Now().Format(time.RFC3339),
	})
}

// HealthCheck godoc
// @Summary Pemeriksaan Kesiapan Service (Readiness Probe)
// @Description Memeriksa koneksi database PostgreSQL dan Redis
// @Tags Health
// @Produce json
// @Success 200 {object} response.SuccessResponse
// @Failure 503 {object} response.ErrorResponse
// @Router /health/ready [get]
func (ctl *HealthController) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "up"
	if ctl.db != nil {
		sqlDB, err := ctl.db.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil {
			dbStatus = "down"
		}
	} else {
		dbStatus = "down"
	}

	redisStatus := "up"
	if ctl.rdb != nil {
		if err := ctl.rdb.Ping(ctx).Err(); err != nil {
			redisStatus = "down"
		}
	} else {
		redisStatus = "down"
	}

	data := gin.H{
		"service":  "auth-service",
		"env":      ctl.env,
		"database": dbStatus,
		"redis":    redisStatus,
		"time":     time.Now().Format(time.RFC3339),
	}

	if dbStatus == "down" || redisStatus == "down" {
		c.JSON(http.StatusServiceUnavailable, response.ErrorResponse{
			Success:   false,
			Message:   "Beberapa dependensi layanan tidak sehat",
			ErrorCode: response.ErrCodeUpstreamUnavailable,
		})
		return
	}

	response.Success(c, http.StatusOK, "Auth service dalam kondisi sehat", data)
}
