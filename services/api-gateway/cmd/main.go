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
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/config"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/health"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/jwks"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/proxy"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/server"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Gagal memuat konfigurasi API Gateway: " + err.Error())
	}

	appLog, err := logger.Init(cfg.AppEnv)
	if err != nil {
		panic("Gagal menginisialisasi logger API Gateway: " + err.Error())
	}
	defer appLog.Sync()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	appLog.Info("Memulai API Gateway PMB...",
		zap.String("port", cfg.AppPort),
		zap.String("env", cfg.AppEnv),
		zap.String("routes_config", cfg.RoutesConfigPath),
	)

	// Inisialisasi Redis Client
	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
		})
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			appLog.Warn("Redis tidak merespon saat start gateway (rate limit/denylist offline): " + err.Error())
		} else {
			appLog.Info("✅ Terhubung ke Redis")
		}
		pingCancel()
	}

	// Inisialisasi JWKS Client
	jwksClient := jwks.NewClient(cfg.JWKSURL, appLog)

	// Inisialisasi Reverse Proxy Handler
	proxyHandler, err := proxy.NewProxyHandler(cfg.Routes, appLog)
	if err != nil {
		appLog.Fatal("Gagal menginisialisasi reverse proxy", zap.Error(err))
	}

	// Inisialisasi Health Handler
	healthHandler := health.NewHealthHandler(cfg.Routes)

	// Inisialisasi Gin Engine dengan urutan middleware PRD 6.3
	r := server.SetupGatewayEngine(cfg, jwksClient, rdb, proxyHandler, healthHandler, appLog)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLog.Info("API Gateway siap melayani request", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Fatal("Gagal menjalankan server API Gateway", zap.Error(err))
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
