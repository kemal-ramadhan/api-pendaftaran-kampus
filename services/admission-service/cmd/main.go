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
	"github.com/pkm-asia/pmb-backend/pkg/outbox"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Muat Konfigurasi
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Gagal memuat konfigurasi: %v\n", err)
		os.Exit(1)
	}

	// 2. Inisialisasi Logger
	log, err := logger.Init(cfg.AppEnv)
	if err != nil {
		fmt.Printf("Gagal inisialisasi logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	log.Info("Memulai PMB Admission Service...", zap.String("env", cfg.AppEnv), zap.Int("port", cfg.AppPort))

	// 3. Inisialisasi Basis Data
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data admission", zap.Error(err))
	}

	// Auto-migrate tabel admission & outbox
	err = db.AutoMigrate(
		&model.Period{},
		&model.AdmissionPath{},
		&model.DocumentRequirement{},
		&model.Quota{},
		&model.Application{},
		&model.ApplicationDocument{},
		&model.ApplicationStatusLog{},
		&outbox.Event{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel admission", zap.Error(err))
	}
	log.Info("Migrasi basis data admission berhasil disiapkan")

	// 4. Jalankan Seeder Data Awal
	ctxSeed, cancelSeed := context.WithTimeout(context.Background(), 10*time.Second)
	if err := seeder.SeedAdmissionData(ctxSeed, db); err != nil {
		log.Warn("Gagal menjalankan seeder data admission", zap.Error(err))
	} else {
		log.Info("Seeder data master admission selesai dijalankan")
	}
	cancelSeed()

	// 5. Inisialisasi Client Inter-Service
	var notifCli internalclient.Client
	if cfg.NotificationServiceURL != "" {
		notifCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.NotificationServiceURL,
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

	var identityCli internalclient.Client
	if cfg.IdentityServiceURL != "" {
		identityCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.IdentityServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	// Inisialisasi Storage Client (MinIO)
	var storageCli storage.Client
	if cfg.MinIOEndpoint != "" {
		sc, sErr := storage.NewClient(storage.Config{
			Endpoint:        cfg.MinIOEndpoint,
			AccessKeyID:     cfg.MinIOAccessKey,
			SecretAccessKey: cfg.MinIOSecretKey,
			UseSSL:          cfg.MinIOUseSSL,
		})
		if sErr != nil {
			log.Warn("Gagal inisialisasi MinIO client:", zap.Error(sErr))
		} else {
			storageCli = sc
			_ = storageCli.EnsureBucket(context.Background(), cfg.DocumentBucket)
		}
	}

	// 6. Inisialisasi Outbox Worker
	outboxRepo := outbox.NewRepository()
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	outboxDispatcher := service.OutboxDispatcher(financeCli)
	outbox.StartWorker(
		outboxCtx,
		db,
		outboxRepo,
		outboxDispatcher,
		time.Duration(cfg.OutboxIntervalSeconds)*time.Second,
		cfg.OutboxBatchSize,
		3,
	)
	log.Info("Background Transactional Outbox worker dimulai")

	// 7. Inisialisasi Repository
	periodRepo := repository.NewPeriodRepository(db)
	pathRepo := repository.NewAdmissionPathRepository(db)
	quotaRepo := repository.NewQuotaRepository(db)
	appRepo := repository.NewApplicationRepository(db)
	docRepo := repository.NewApplicationDocumentRepository(db)
	reqRepo := repository.NewDocumentRequirementRepository(db)

	// 8. Inisialisasi Service
	admissionSvc := service.NewAdmissionService(
		periodRepo,
		pathRepo,
		quotaRepo,
		appRepo,
		docRepo,
		outboxRepo,
		notifCli,
		identityCli,
		financeCli,
		storageCli,
		*cfg,
	)

	// 9. Inisialisasi Controllers
	healthCtrl := controller.NewHealthController(db)
	publicCtrl := controller.NewPublicController(admissionSvc)
	appCtrl := controller.NewApplicationController(admissionSvc)
	adminCtrl := controller.NewAdminController(admissionSvc, periodRepo, pathRepo, quotaRepo, reqRepo)
	internalCtrl := controller.NewInternalController(admissionSvc)

	// 10. Setup Router Gin
	r := route.SetupRouter(
		route.RouterConfig{
			GatewaySecret: cfg.GatewaySharedSecret,
			InternalToken: cfg.InternalToken,
		},
		healthCtrl,
		publicCtrl,
		appCtrl,
		adminCtrl,
		internalCtrl,
	)

	// 11. Jalankan HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Server Admission Service berjalan", zap.Int("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server Admission Service berhenti dengan error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Menerima sinyal terminasi, mematikan server secara graceful...", zap.String("signal", sig.String()))

	outboxCancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server dipaksa berhenti:", zap.Error(err))
	}

	log.Info("Admission Service berhasil dihentikan dengan aman.")
}
