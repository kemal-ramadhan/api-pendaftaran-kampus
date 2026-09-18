package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8000"
	}

	appLog, err := logger.Init(appEnv)
	if err != nil {
		panic("Gagal menginisialisasi logger API Gateway: " + err.Error())
	}
	defer appLog.Sync()

	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(logger.GinRecovery(appLog))
	r.Use(logger.GinLogger(appLog))

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "API Gateway healthy", gin.H{
			"status":  "up",
			"service": "api-gateway",
			"env":     appEnv,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	srv := &http.Server{
		Addr:         ":" + appPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLog.Info("Memulai API Gateway", zap.String("port", appPort), zap.String("env", appEnv))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Fatal("Gagal menjalankan API Gateway", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLog.Info("Mematikan API Gateway secara anggun (graceful shutdown)...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLog.Fatal("API Gateway dipaksa berhenti", zap.Error(err))
	}

	appLog.Info("API Gateway berhasil dihentikan")
}
