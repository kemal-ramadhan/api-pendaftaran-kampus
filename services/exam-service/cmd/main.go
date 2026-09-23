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
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/service"
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

	log.Info("Memulai PMB Exam Service...", zap.String("env", cfg.AppEnv), zap.String("port", cfg.Port))

	// 3. Inisialisasi Basis Data
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data exam", zap.Error(err))
	}

	// Auto-migrate tabel exam
	err = db.AutoMigrate(
		&model.ExamSession{},
		&model.SessionExaminer{},
		&model.Participant{},
		&model.ExamCard{},
	)
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel exam", zap.Error(err))
	}
	log.Info("Migrasi basis data exam berhasil disiapkan")

	// 4. Jalankan Seeder
	ctxSeed, cancelSeed := context.WithTimeout(context.Background(), 10*time.Second)
	if err := seeder.SeedExamSessions(ctxSeed, db); err != nil {
		log.Warn("Gagal menjalankan seeder exam session", zap.Error(err))
	}
	cancelSeed()

	// 5. Inisialisasi MinIO Storage Client
	var storageCli storage.Client
	if cfg.MinIOEndpoint != "" {
		sCli, sErr := storage.NewClient(storage.Config{
			Endpoint:        cfg.MinIOEndpoint,
			AccessKeyID:     cfg.MinIOAccessKey,
			SecretAccessKey: cfg.MinIOSecretKey,
			UseSSL:          cfg.MinIOUseSSL,
		})
		if sErr != nil {
			log.Warn("Gagal menghubungkan MinIO storage client", zap.Error(sErr))
		} else {
			storageCli = sCli
			ctxInit, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
			_ = storageCli.EnsureBucket(ctxInit, cfg.ExamCardBucket)
			cancelInit()
			log.Info("MinIO storage client berhasil diinisialisasi", zap.String("bucket", cfg.ExamCardBucket))
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

	var identityCli internalclient.Client
	if cfg.IdentityServiceURL != "" {
		identityCli = internalclient.NewClient(internalclient.Config{
			BaseURL:       cfg.IdentityServiceURL,
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

	// 7. Inisialisasi Repository & Service Layer
	sessionRepo := repository.NewSessionRepository(db)
	participantRepo := repository.NewParticipantRepository(db)

	examSvc := service.NewExamService(
		cfg,
		sessionRepo,
		participantRepo,
		storageCli,
		admissionCli,
		identityCli,
		notificationCli,
	)

	// 8. Inisialisasi Controller
	healthCtrl := controller.NewHealthController()
	sessionCtrl := controller.NewSessionController(examSvc)
	participantCtrl := controller.NewParticipantController(examSvc)
	studentCtrl := controller.NewStudentController(examSvc)
	examinerCtrl := controller.NewExaminerController(examSvc)
	internalCtrl := controller.NewInternalController(examSvc)

	// 9. Inisialisasi HTTP Router
	routerCfg := route.RouterConfig{
		GatewaySecret: cfg.GatewaySharedKey,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment(),
	}
	r := route.SetupRouter(
		routerCfg,
		healthCtrl,
		sessionCtrl,
		participantCtrl,
		studentCtrl,
		examinerCtrl,
		internalCtrl,
	)

	// 10. Start Server dengan Graceful Shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Exam Service siap melayani permintaan", zap.String("port", cfg.Port))
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

	log.Info("PMB Exam Service telah berhenti dengan bersih")
}
