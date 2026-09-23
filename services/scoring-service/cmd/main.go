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
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/service"
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

	log.Info("Memulai PMB Scoring Service...", zap.String("env", cfg.AppEnv), zap.String("port", cfg.Port))

	// 3. Inisialisasi Basis Data
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data scoring", zap.Error(err))
	}

	// Auto-migrate tabel scoring
	err = db.AutoMigrate(
		&model.ScoreComponent{},
		&model.Score{},
		&model.ScoreSummary{},
		&model.ProgramLock{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel scoring", zap.Error(err))
	}
	log.Info("Migrasi basis data scoring berhasil disiapkan")

	// 4. Jalankan Seeder
	ctxSeed, cancelSeed := context.WithTimeout(context.Background(), 10*time.Second)
	if err := seeder.SeedScoreComponents(ctxSeed, db); err != nil {
		log.Warn("Gagal menjalankan seeder score components", zap.Error(err))
	}
	cancelSeed()

	// 5. Inisialisasi Client Inter-Service
	var admissionCli internalclient.Client
	if cfg.AdmissionServiceURL != "" {
		admissionCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.AdmissionServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var examCli internalclient.Client
	if cfg.ExamServiceURL != "" {
		examCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.ExamServiceURL,
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

	// 6. Inisialisasi Exporter, Repository & Service Layer
	excelExp := exporter.NewExcelExporter()
	compRepo := repository.NewComponentRepository(db)
	scoreRepo := repository.NewScoreRepository(db)

	scoringSvc := service.NewScoringService(
		cfg,
		compRepo,
		scoreRepo,
		excelExp,
		admissionCli,
		examCli,
		identityCli,
	)

	// 7. Inisialisasi Controller
	healthCtrl := controller.NewHealthController()
	compCtrl := controller.NewComponentController(scoringSvc)
	examinerCtrl := controller.NewExaminerController(scoringSvc)
	progCtrl := controller.NewProgramController(scoringSvc)
	internalCtrl := controller.NewInternalController(scoringSvc)

	// 8. Inisialisasi HTTP Router
	routerCfg := route.RouterConfig{
		GatewaySecret: cfg.GatewaySharedKey,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment(),
	}
	r := route.SetupRouter(
		routerCfg,
		healthCtrl,
		compCtrl,
		examinerCtrl,
		progCtrl,
		internalCtrl,
	)

	// 9. Start Server dengan Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Scoring Service siap melayani permintaan", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Gagal menjalankan HTTP server", zap.Error(err))
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Menerima sinyal mematikan service, melakukan graceful shutdown...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("Terjadi kesalahan saat mematikan HTTP server", zap.Error(err))
	}

	log.Info("PMB Scoring Service telah berhenti dengan bersih")
}
