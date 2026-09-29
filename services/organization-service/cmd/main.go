package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	pkglogger "github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/service"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Gagal memuat konfigurasi: %v\n", err)
		os.Exit(1)
	}

	appLog, err := pkglogger.Init(cfg.AppEnv)
	if err != nil {
		fmt.Printf("Gagal inisialisasi logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = appLog.Sync() }()

	appLog.Info("Memulai "+cfg.AppName, zap.String("env", cfg.AppEnv), zap.String("port", cfg.AppPort))

	// Koneksi Database
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		appLog.Warn("Database belum dapat dihubungi saat startup", zap.Error(err))
	} else {
		// AutoMigrate entitas
		if err := db.AutoMigrate(&model.Faculty{}, &model.StudyProgram{}, &model.InstitutionProfile{}); err != nil {
			appLog.Error("Gagal menjalankan migrasi schema", zap.Error(err))
		} else {
			appLog.Info("Migrasi basis data organization selesai")
			// Jalankan seeder
			if err := seeder.Seed(context.Background(), db, appLog); err != nil {
				appLog.Error("Gagal menjalankan seeder organization", zap.Error(err))
			}
		}
	}

	// Inisialisasi Dependensi
	facultyRepo := repository.NewFacultyRepository(db)
	studyProgramRepo := repository.NewStudyProgramRepository(db)
	institutionProfileRepo := repository.NewInstitutionProfileRepository(db)

	orgService := service.NewOrganizationService(facultyRepo, studyProgramRepo, institutionProfileRepo)

	pubCtl := controller.NewPublicController(orgService)
	admCtl := controller.NewAdminController(orgService)
	internalCtl := controller.NewInternalController(orgService)
	healthCtl := controller.NewHealthController(db)

	// Mode Gin
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(pkglogger.GinRecovery(appLog))
	r.Use(requestid.Middleware())
	r.Use(pkglogger.GinLogger(appLog))

	route.SetupRouter(r, pubCtl, admCtl, internalCtl, healthCtl, cfg.GatewaySharedSecret, cfg.InternalToken)

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: r,
	}

	go func() {
		appLog.Info("HTTP server mendengarkan pada port :" + cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLog.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLog.Info("Menutup organization-service secara graceful...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLog.Error("Gagal shutdown HTTP server", zap.Error(err))
	}
	appLog.Info("Organization service telah berhenti")
}
