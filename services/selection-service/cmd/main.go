package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/service"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Muat Konfigurasi
	cfg := config.LoadConfig()

	// 2. Inisialisasi Logger
	log, err := logger.Init(cfg.AppEnv)
	if err != nil {
		fmt.Printf("Gagal inisialisasi logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	log.Info("Memulai PMB Selection Service...", zap.String("env", cfg.AppEnv), zap.String("port", cfg.Port))

	// 3. Inisialisasi Basis Data
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s client_encoding=UTF8",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data selection", zap.Error(err))
	}

	// Auto-migrate tabel selection
	err = db.AutoMigrate(
		&model.Recommendation{},
		&model.RecommendationItem{},
		&model.Announcement{},
		&model.Result{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel selection", zap.Error(err))
	}
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_recommendations_lookup ON recommendations(period_id, path_id, study_program_code)`).Error
	log.Info("Migrasi basis data selection berhasil disiapkan")

	// 4. Inisialisasi Client Inter-Service
	var scoringCli internalclient.Client
	if cfg.ScoringServiceURL != "" {
		scoringCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.ScoringServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var admissionCli internalclient.Client
	if cfg.AdmissionServiceURL != "" {
		admissionCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.AdmissionServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var financeCli internalclient.Client
	if cfg.FinanceServiceURL != "" {
		financeCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.FinanceServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var notificationCli internalclient.Client
	if cfg.NotificationServiceURL != "" {
		notificationCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.NotificationServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var identityCli internalclient.Client
	if cfg.IdentityServiceURL != "" {
		identityCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.IdentityServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	// 5. Inisialisasi Repository & Service Layer
	recRepo := repository.NewRecommendationRepository(db)
	annRepo := repository.NewAnnouncementRepository(db)
	resultRepo := repository.NewResultRepository(db)

	selectionSvc := service.NewSelectionService(
		cfg,
		recRepo,
		annRepo,
		resultRepo,
		scoringCli,
		admissionCli,
		financeCli,
		notificationCli,
		identityCli,
	)

	// 6. Inisialisasi Controller
	healthCtrl := controller.NewHealthController()
	progCtrl := controller.NewProgramController(selectionSvc)
	approvalCtrl := controller.NewApprovalController(selectionSvc)
	announcementCtrl := controller.NewAnnouncementController(selectionSvc)
	studentCtrl := controller.NewStudentController(selectionSvc)
	internalCtrl := controller.NewInternalController(resultRepo, selectionSvc)

	// 7. Inisialisasi HTTP Router
	routerCfg := route.RouterConfig{
		GatewaySecret: cfg.GatewaySharedKey,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment(),
	}
	r := route.SetupRouter(
		routerCfg,
		healthCtrl,
		progCtrl,
		approvalCtrl,
		announcementCtrl,
		studentCtrl,
		internalCtrl,
	)

	// 8. Jalankan HTTP Server dengan Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Selection Service mendengarkan pada port " + cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Gagal menjalankan Selection Service HTTP server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Menerima sinyal terminasi, mematikan Selection Service secara graceful...", zap.String("signal", sig.String()))

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("Gagal shutdown graceful Selection Service", zap.Error(err))
	} else {
		log.Info("Selection Service berhasil dimatikan dengan aman")
	}
}
