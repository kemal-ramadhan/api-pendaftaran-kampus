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

	"github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/sender"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/service"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/template"
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

	log.Info("Memulai PMB Notification Service...", zap.String("env", cfg.AppEnv), zap.Int("port", cfg.AppPort))

	// 3. Inisialisasi Basis Data
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal terhubung ke basis data notification", zap.Error(err))
	}

	// Auto-migrate tabel notifikasi
	err = db.AutoMigrate(&model.NotificationLog{})
	if err != nil {
		log.Fatal("Gagal migrasi skema tabel notification", zap.Error(err))
	}
	log.Info("Migrasi basis data notification berhasil disiapkan")

	// 4. Inisialisasi Sender dan Engine
	emailSnd := sender.NewEmailSender(cfg)
	waSnd := sender.NewWhatsAppSender(cfg)
	tmplEngine := template.NewTemplateEngine()

	// 5. Inisialisasi Repository & Service
	notifRepo := repository.NewNotificationRepository(db)
	notifSvc := service.NewNotificationService(notifRepo, emailSnd, waSnd, tmplEngine)

	// 6. Inisialisasi & Jalankan Background Worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	worker := service.NewWorker(
		notifSvc,
		time.Duration(cfg.WorkerIntervalSeconds)*time.Second,
		cfg.WorkerBatchSize,
	)
	go worker.Start(workerCtx)

	// 7. Inisialisasi Controller
	healthCtrl := controller.NewHealthController()
	notifCtrl := controller.NewNotificationController(notifSvc)

	// 8. Setup Router
	r := route.SetupRouter(
		route.RouterConfig{
			InternalToken: cfg.InternalToken,
		},
		healthCtrl,
		notifCtrl,
	)

	// 9. Jalankan HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Server Notification Service berjalan", zap.Int("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server Notification Service berhenti dengan error", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("Menerima sinyal terminasi, mematikan server secara graceful...", zap.String("signal", sig.String()))

	workerCancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server dipaksa berhenti:", zap.Error(err))
	}

	log.Info("Notification Service berhasil dihentikan dengan aman.")
}
