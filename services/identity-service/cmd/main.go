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
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/service"
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

	log.Info("Memulai PMB Identity Service...", zap.String("env", cfg.AppEnv), zap.Int("port", cfg.AppPort))

	// 3. Inisialisasi Basis Data
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data identity", zap.Error(err))
	}

	// Auto-migrate tabel identity
	err = db.AutoMigrate(
		&model.Person{},
		&model.Address{},
		&model.Guardian{},
		&model.EducationHistory{},
		&model.KTPScan{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel identity", zap.Error(err))
	}
	log.Info("Migrasi basis data identity berhasil disiapkan")

	// 4. Inisialisasi Kunci Kriptografi
	encryptionKey := crypto.DeriveKey(cfg.IdentityEncryptionKey)

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
			log.Warn("Gagal menghubungkan MinIO storage client, fitur penyimpanan foto/KTP mungkin terbatas", zap.Error(sErr))
		} else {
			storageCli = sCli
			ctxInit, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
			_ = storageCli.EnsureBucket(ctxInit, "pmb-identity-photos")
			_ = storageCli.EnsureBucket(ctxInit, "pmb-identity-ktp")
			cancelInit()
			log.Info("MinIO storage client berhasil diinisialisasi")
		}
	}

	// 6. Inisialisasi Internal OCR Client
	var ocrCli internalclient.Client
	if cfg.OCRServiceURL != "" {
		ocrCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.OCRServiceURL,
			InternalToken: cfg.InternalToken,
			Timeout:       30 * time.Second, // Timeout lebih besar untuk proses OCR
		})
		log.Info("Internal OCR client dikonfigurasi", zap.String("ocr_url", cfg.OCRServiceURL))
	}

	// 7. Inisialisasi Repository
	personRepo := repository.NewPersonRepository(db)
	addressRepo := repository.NewAddressRepository(db)
	guardianRepo := repository.NewGuardianRepository(db)
	educationRepo := repository.NewEducationRepository(db)
	ktpScanRepo := repository.NewKTPScanRepository(db)

	// 8. Inisialisasi Service
	identitySvc := service.NewIdentityService(
		personRepo,
		addressRepo,
		guardianRepo,
		educationRepo,
		storageCli,
		encryptionKey,
		"pmb-identity-photos",
	)
	completenessSvc := service.NewCompletenessService(personRepo)
	ktpScanSvc := service.NewKTPScanService(
		ktpScanRepo,
		identitySvc,
		storageCli,
		ocrCli,
		"pmb-identity-ktp",
	)

	// 9. Inisialisasi Controller
	healthCtrl := controller.NewHealthController(db)
	userCtrl := controller.NewUserController(identitySvc, completenessSvc, ktpScanSvc)
	adminCtrl := controller.NewAdminController(identitySvc, completenessSvc)
	internalCtrl := controller.NewInternalController(identitySvc, completenessSvc)

	// 10. Setup Router Gin
	r := route.SetupRouter(
		route.RouterConfig{
			GatewaySecret: cfg.GatewaySharedSecret,
			InternalToken: cfg.InternalToken,
		},
		healthCtrl,
		userCtrl,
		adminCtrl,
		internalCtrl,
	)

	// 11. Jalankan HTTP Server dengan Graceful Shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Server Identity Service berjalan", zap.Int("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server Identity Service berhenti dengan error", zap.Error(err))
		}
	}()

	// Menunggu sinyal OS untuk graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Menerima sinyal terminasi, mematikan server secara graceful...", zap.String("signal", sig.String()))

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server dipaksa berhenti:", zap.Error(err))
	}

	log.Info("Identity Service berhasil dihentikan dengan aman.")
}
