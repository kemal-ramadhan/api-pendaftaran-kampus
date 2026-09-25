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
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/provider"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/service"
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

	log.Info("Memulai PMB Finance Service...", zap.String("env", cfg.AppEnv), zap.Int("port", cfg.AppPort))

	// 3. Inisialisasi Basis Data
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data finance", zap.Error(err))
	}

	// Auto-migrate tabel finance
	err = db.AutoMigrate(
		&model.FeeScheme{},
		&model.Invoice{},
		&model.Payment{},
		&model.Receipt{},
		&model.PaymentWebhookLog{},
		&model.InstallmentRequest{},
		&model.Installment{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel finance", zap.Error(err))
	}
	log.Info("Migrasi basis data finance berhasil disiapkan")

	// 4. Jalankan Seeder
	ctxSeed, cancelSeed := context.WithTimeout(context.Background(), 10*time.Second)
	if err := seeder.SeedFeeSchemes(ctxSeed, db); err != nil {
		log.Warn("Gagal menjalankan seeder skema tarif biaya", zap.Error(err))
	}
	cancelSeed()

	// 5. Inisialisasi MinIO Storage Client
	var storageCli storage.Client
	if cfg.MinioEndpoint != "" {
		sCli, sErr := storage.NewClient(storage.Config{
			Endpoint:        cfg.MinioEndpoint,
			AccessKeyID:     cfg.MinioAccessKey,
			SecretAccessKey: cfg.MinioSecretKey,
			UseSSL:          cfg.MinioUseSSL,
			PublicEndpoint:  cfg.MinioPublicEndpoint,
		})
		if sErr != nil {
			log.Warn("Gagal menghubungkan MinIO storage client", zap.Error(sErr))
		} else {
			storageCli = sCli
			ctxInit, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
			_ = storageCli.EnsureBucket(ctxInit, "pmb-finance-proofs")
			_ = storageCli.EnsureBucket(ctxInit, "pmb-finance-receipts")
			cancelInit()
			log.Info("MinIO storage client berhasil diinisialisasi")
		}
	}

	// 6. Inisialisasi Client Inter-Service
	var admissionCli internalclient.Client
	if cfg.AdmissionServiceURL != "" {
		admissionCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.AdmissionServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	var notifCli internalclient.Client
	if cfg.NotificationServiceURL != "" {
		notifCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.NotificationServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}

	// 7. Inisialisasi Payment Provider & Exporter
	var paymentProvider provider.PaymentProvider
	if cfg.PaymentProvider == "duitku" {
		dProvider, dErr := provider.NewDuitkuProvider(provider.DuitkuConfig{
			Env:           cfg.DuitkuEnv,
			MerchantCode:  cfg.DuitkuMerchantCode,
			APIKey:        cfg.DuitkuApiKey,
			ExpiryMinutes: cfg.DuitkuExpiryMinutes,
			CallbackURL:   cfg.DuitkuCallbackURL,
			ReturnURL:     cfg.DuitkuReturnURL,
		})
		if dErr != nil {
			log.Fatal("Gagal inisialisasi Duitku payment provider", zap.Error(dErr))
		}
		paymentProvider = dProvider
		log.Info("Menggunakan Payment Provider: Duitku POP", zap.String("env", cfg.DuitkuEnv))
	} else {
		paymentProvider = provider.NewMockProvider()
		log.Info("Menggunakan Payment Provider: MockProvider (Development/Testing)")
	}
	reportExp := exporter.NewReportExporter()

	// 8. Inisialisasi Repository
	invRepo := repository.NewInvoiceRepository(db)
	payRepo := repository.NewPaymentRepository(db)
	rcRepo := repository.NewReceiptRepository(db)
	webhookLogRepo := repository.NewWebhookLogRepository(db)
	installmentRepo := repository.NewInstallmentRepository(db)
	reportRepo := repository.NewReportRepository(db)
	feeSchemeRepo := repository.NewFeeSchemeRepository(db)

	// 9. Inisialisasi Service
	invSvc := service.NewInvoiceService(invRepo)
	paySvc := service.NewPaymentService(
		payRepo,
		invRepo,
		rcRepo,
		webhookLogRepo,
		paymentProvider,
		storageCli,
		admissionCli,
		notifCli,
		cfg,
	)
	paySvc.SetInstallmentRepo(installmentRepo)

	installmentSvc := service.NewInstallmentService(installmentRepo, invRepo, feeSchemeRepo, notifCli)
	reportSvc := service.NewReportService(reportRepo, reportExp)

	// 10. Inisialisasi Controller
	healthCtrl := controller.NewHealthController(db)
	invCtrl := controller.NewInvoiceController(invSvc)
	payCtrl := controller.NewPaymentController(paySvc)
	rcCtrl := controller.NewReceiptController(paySvc)
	webhookCtrl := controller.NewWebhookController(paySvc)
	devCtrl := controller.NewDevController(paySvc, cfg)
	adminCtrl := controller.NewAdminController(paySvc, feeSchemeRepo)
	internalCtrl := controller.NewInternalController(invSvc)
	installmentCtrl := controller.NewInstallmentController(installmentSvc)
	reportCtrl := controller.NewReportController(reportSvc, installmentSvc)

	// 11. Setup Router Gin
	r := route.SetupRouter(
		route.RouterConfig{
			GatewaySecret: cfg.GatewaySharedSecret,
			InternalToken: cfg.InternalToken,
			IsDevelopment: cfg.IsDevelopment(),
		},
		healthCtrl,
		invCtrl,
		payCtrl,
		rcCtrl,
		webhookCtrl,
		devCtrl,
		adminCtrl,
		internalCtrl,
		installmentCtrl,
		reportCtrl,
	)

	// 12. Jalankan HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Server Finance Service berjalan", zap.Int("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server Finance Service berhenti dengan error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Menerima sinyal terminasi, mematikan server secara graceful...", zap.String("signal", sig.String()))

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server dipaksa berhenti:", zap.Error(err))
	}

	log.Info("Finance Service berhasil dihentikan dengan aman.")
}
