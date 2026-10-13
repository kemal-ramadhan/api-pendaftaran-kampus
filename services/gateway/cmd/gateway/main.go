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
	pkglogger "github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	authconfig "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	authcontroller "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/controller"
	authlimiter "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/limiter"
	authmailer "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/mailer"
	authmiddleware "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/middleware"
	authmigrations "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/migrations"
	authrepo "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/repository"
	authseeder "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/seeder"
	authservice "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/service"
	authtoken "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
	gatewayconfig "github.com/pkm-asia/pmb-backend/services/gateway/internal/gatewayconfig"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/health"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/jwks"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/proxy"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/server"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	gwCfg, err := gatewayconfig.LoadConfig()
	if err != nil {
		panic("Gagal memuat konfigurasi API Gateway: " + err.Error())
	}
	authCfg := authconfig.LoadConfig()

	appLog, err := pkglogger.Init(gwCfg.AppEnv)
	if err != nil {
		panic("Gagal menginisialisasi logger API Gateway: " + err.Error())
	}
	defer appLog.Sync()

	if gwCfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	appLog.Info("Memulai Gateway PMB (Gateway + Auth)...",
		zap.String("port", gwCfg.AppPort),
		zap.String("env", gwCfg.AppEnv),
		zap.String("routes_config", gwCfg.RoutesConfigPath),
	)

	// 1. Basis Data PostgreSQL pmb_auth
	var db *gorm.DB
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = authconfig.ConnectDB(authCfg)
		if err == nil {
			break
		}
		appLog.Warn("Menunggu koneksi DB pmb_auth...", zap.Int("attempt", attempt), zap.Error(err))
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		appLog.Fatal("Koneksi DB pmb_auth belum tersedia setelah retry: " + err.Error())
	}
	appLog.Info("✅ Terhubung ke PostgreSQL pmb_auth")

	// Migrasi & Seeder Auth
	migCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := authmigrations.RunMigrations(migCtx, db, appLog); err != nil {
		appLog.Fatal("Gagal menjalankan migrasi auth", zap.Error(err))
	}
	_ = authseeder.SeedRoles(migCtx, db, appLog)
	_ = authseeder.SeedPermissions(migCtx, db, appLog)
	_ = authseeder.SeedSuperadmin(migCtx, db, authCfg, appLog)
	migCancel()

	// 2. Redis Client
	var rdb *redis.Client
	if gwCfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     gwCfg.RedisAddr,
			Password: gwCfg.RedisPassword,
		})
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			appLog.Warn("Redis tidak merespon saat start gateway: " + err.Error())
		} else {
			appLog.Info("✅ Terhubung ke Redis")
		}
		pingCancel()
	}

	// 3. Auth Token Manager & Layer Servis
	tokenMgr, err := authtoken.NewTokenManager(authCfg, appLog)
	if err != nil {
		appLog.Fatal("Gagal menginisialisasi token manager", zap.Error(err))
	}
	sessionStore := authtoken.NewSessionStore(rdb)

	userRepo := authrepo.NewUserRepository(db)
	roleRepo := authrepo.NewRoleRepository(db)
	otpRepo := authrepo.NewOTPRepository(db)
	resetRepo := authrepo.NewPasswordResetRepository(db)
	auditRepo := authrepo.NewAuditLogRepository(db)

	rateLimiter := authlimiter.NewRateLimiter(rdb)
	mailService := authmailer.NewMailer(authCfg, appLog)

	authSvc := authservice.NewAuthService(
		authCfg,
		userRepo,
		roleRepo,
		otpRepo,
		resetRepo,
		auditRepo,
		tokenMgr,
		sessionStore,
		rateLimiter,
		mailService,
		appLog,
	)

	adminSvc := authservice.NewAdminService(
		authCfg,
		userRepo,
		roleRepo,
		resetRepo,
		auditRepo,
		sessionStore,
		rateLimiter,
		mailService,
		nil, // Inter-service client ke organization jika dipanggil
		appLog,
	)

	authCtrl := authcontroller.NewAuthController(authSvc, tokenMgr)
	adminCtrl := authcontroller.NewAdminController(adminSvc)
	internalCtrl := authcontroller.NewInternalController(adminSvc)
	authHealthCtrl := authcontroller.NewHealthController(db, rdb, authCfg.AppEnv)

	// 4. Local JWKS Client (Validasi JWT lokal dari RSA public key sendiri)
	jwksClient := jwks.NewClient(gwCfg.JWKSURL, appLog)
	if tokenMgr.PublicKey() != nil {
		jwksClient.SetStaticKey(authCfg.JWTKid, tokenMgr.PublicKey())
		jwksClient.SetStaticKey("pmb-auth-1", tokenMgr.PublicKey())
	}

	// 5. Reverse Proxy Handler
	proxyHandler, err := proxy.NewProxyHandler(gwCfg.Routes, appLog)
	if err != nil {
		appLog.Fatal("Gagal menginisialisasi reverse proxy", zap.Error(err))
	}

	healthHandler := health.NewHealthHandler(gwCfg.Routes)

	// 6. Gin Engine Utama (Port :8000)
	mainRouter := server.SetupGatewayEngine(
		gwCfg,
		jwksClient,
		rdb,
		proxyHandler,
		healthHandler,
		authCtrl,
		adminCtrl,
		tokenMgr,
		sessionStore,
		appLog,
	)

	// 7. Gin Engine Internal (Port :8081 - inter-service internal endpoints)
	internalRouter := gin.New()
	internalRouter.Use(pkglogger.GinRecovery(appLog))
	internalRouter.Use(requestid.Middleware())
	internalRouter.Use(pkglogger.GinLogger(appLog))

	internalRouter.GET("/health", authHealthCtrl.HealthCheck)
	internalGroup := internalRouter.Group("/internal")
	internalGroup.Use(authmiddleware.InternalAuthMiddleware(authCfg.InternalToken))
	{
		internalGroup.GET("/users/:id", internalCtrl.GetUser)
		internalGroup.POST("/users/batch", internalCtrl.GetBatchUsers)
		internalGroup.POST("/users/:id/roles", internalCtrl.AddUserRole)
		internalGroup.POST("/users/:id/verify-password", internalCtrl.VerifyPassword)
		internalGroup.DELETE("/users/:id/roles/:code", internalCtrl.RemoveUserRole)
	}

	mainServer := &http.Server{
		Addr:         ":" + gwCfg.AppPort,
		Handler:      mainRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	internalPort := os.Getenv("INTERNAL_PORT")
	if internalPort == "" {
		internalPort = "8081"
	}
	internalServer := &http.Server{
		Addr:         ":" + internalPort,
		Handler:      internalRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLog.Info("Main Gateway siap melayani request", zap.String("addr", mainServer.Addr))
		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Fatal("Gagal menjalankan main server Gateway", zap.Error(err))
		}
	}()

	go func() {
		appLog.Info("Internal Gateway siap melayani inter-service request", zap.String("addr", internalServer.Addr))
		if err := internalServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Warn("Internal server stopped", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLog.Info("Mematikan Gateway secara anggun (graceful shutdown)...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	_ = mainServer.Shutdown(shutdownCtx)
	_ = internalServer.Shutdown(shutdownCtx)

	appLog.Info("Gateway berhasil dihentikan")
}
