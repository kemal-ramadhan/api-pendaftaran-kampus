package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	_ "github.com/iamhexagoninc/go-jatidiri-app/docs"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/middleware"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/route"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
)

// @title Jatidiri Auth Service API
// @version 1.0
// @description Authentication service for Jatidiri.app
// @BasePath /

// @host localhost:8080
// @schemes http

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
func main() {
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	config.InitLogger()
	if err := config.Logger.Sync(); err != nil {
	}
	defer config.Logger.Sync()

	config.ConnectDB()
	config.ConnectRedis()

	config.InitTracer()
	defer config.ShutdownTracer(context.Background())

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleware())

	r.Use(otelgin.Middleware("jatidiri-auth-service"))

	prom := ginprom.New(
		ginprom.Engine(r),
		ginprom.Subsystem("jatidiri_auth_service"),
		ginprom.Path("/metrics"),
	)
	r.Use(prom.Instrument())

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecureHeaders())
	// r.Use(middleware.GatewayAuthMiddleware())

	r.Use(middleware.RateLimiter(middleware.RateLimitOptions{
		Prefix:    "rl:global",
		Requests:  100,             // Maks 100 request
		Window:    1 * time.Minute, // Per 1 menit
		UseIPOnly: true,            // Per-IP
	}))

	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	})

	route.SetupRoutes(r)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		config.Logger.Info("Jatidiri Auth Service running",
			zap.String("port", port),
			zap.String("env", os.Getenv("APP_ENV")),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	config.Logger.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		config.Logger.Fatal("Forced to shutdown", zap.Error(err))
	}

	config.Logger.Info("Server exited cleanly")
}
