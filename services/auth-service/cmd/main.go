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
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/limiter"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/mailer"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/migrations"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/service"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	_ "github.com/pkm-asia/pmb-backend/services/auth-service/docs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// @title PMB Auth Service API
// @version 1.0
// @description Layanan Autentikasi dan Manajemen Pengguna Sistem PMB
// @contact.name Tim Pengembang PMB
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	appLog, err := logger.Init(cfg.AppEnv)
	if err != nil {
		panic("Gagal menginisialisasi logger: " + err.Error())
	}
	defer appLog.Sync()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	appLog.Info("Memulai Auth Service...",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.AppPort),
	)

	// Inisialisasi Database PostgreSQL dengan retry
	var db *gorm.DB
	for attempt := 1; attempt <= 15; attempt++ {
		db, err = config.ConnectDB(cfg)
		if err == nil {
			break
		}
		appLog.Warn("Menunggu koneksi DB...", zap.Int("attempt", attempt), zap.Error(err))
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		appLog.Warn("Koneksi DB belum tersedia saat start: " + err.Error())
	} else {
		appLog.Info("✅ Terhubung ke PostgreSQL")
		// Jalankan Migrasi & Seeder
		migCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := migrations.RunMigrations(migCtx, db, appLog); err != nil {
			migCancel()
			appLog.Fatal("Gagal menjalankan migrasi skema database", zap.Error(err))
		}
		if err := seeder.SeedRoles(migCtx, db, appLog); err != nil {
			appLog.Warn("Gagal seed role", zap.Error(err))
		}
		if err := seeder.SeedSuperadmin(migCtx, db, cfg, appLog); err != nil {
			appLog.Warn("Gagal seed superadmin", zap.Error(err))
		}
		migCancel()
	}

	// Inisialisasi Redis Client
	rdb, err := config.ConnectRedis(cfg)
	if err != nil {
		appLog.Warn("Koneksi Redis belum tersedia saat start (bisa berjalan di test/container offline): " + err.Error())
	} else {
		appLog.Info("✅ Terhubung ke Redis")
	}

	// Inisialisasi Token Manager & Session Store
	tokenMgr, err := token.NewTokenManager(cfg, appLog)
	if err != nil {
		appLog.Fatal("Gagal menginisialisasi token manager", zap.Error(err))
	}
	sessionStore := token.NewSessionStore(rdb)

	// Inisialisasi Layer Repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	resetRepo := repository.NewPasswordResetRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)

	// Inisialisasi Rate Limiter & Mailer
	rateLimiter := limiter.NewRateLimiter(rdb)
	mailService := mailer.NewMailer(cfg, appLog)

	// Inisialisasi Business Service
	authService := service.NewAuthService(
		cfg,
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

	// Inisialisasi HTTP Client ke Organization Service untuk validasi scope
	orgClient := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.OrganizationServiceURL,
		InternalToken: cfg.InternalToken,
		Timeout:       5 * time.Second,
	})

	adminService := service.NewAdminService(
		cfg,
		userRepo,
		roleRepo,
		resetRepo,
		auditRepo,
		sessionStore,
		rateLimiter,
		mailService,
		orgClient,
		appLog,
	)

	// Inisialisasi Controller
	authCtrl := controller.NewAuthController(authService, tokenMgr)
	adminCtrl := controller.NewAdminController(adminService)
	internalCtrl := controller.NewInternalController(adminService)
	healthCtrl := controller.NewHealthController(db, rdb, cfg.AppEnv)

	// Inisialisasi Gin Engine
	r := gin.New()
	r.Use(requestid.Middleware())
	r.Use(logger.GinRecovery(appLog))
	r.Use(logger.GinLogger(appLog))

	// Setup Routes
	route.SetupRoutes(r, authCtrl, adminCtrl, internalCtrl, healthCtrl, tokenMgr, sessionStore, cfg.InternalToken, cfg.GatewaySharedSecret)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLog.Info("Auth Service siap menerima request", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Fatal("Gagal menjalankan Auth Service", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLog.Info("Mematikan Auth Service secara anggun (graceful shutdown)...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLog.Fatal("Auth Service dipaksa berhenti", zap.Error(err))
	}

	appLog.Info("Auth Service berhasil dihentikan")
}
