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
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/client"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/config"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/job"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/lock"
	platformredis "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/redis"
	platformstorage "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/storage"

	// 10 Modules
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/exam"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/nim"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/selection"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/student"

	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		panic("Gagal memuat konfigurasi pmb-service: " + err.Error())
	}

	appLog, err := pkglogger.Init(cfg.AppEnv)
	if err != nil {
		panic("Gagal inisialisasi logger: " + err.Error())
	}
	defer appLog.Sync()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	appLog.Info("Memulai Modular Monolith PMB Service...",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.AppEnv),
		zap.String("port", cfg.AppPort),
	)

	// 1. Basis Data PostgreSQL (database: pmb_app)
	baseDB, err := platformdb.ConnectDB(cfg, appLog)
	if err != nil {
		appLog.Fatal("Gagal koneksi basis data pmb_app", zap.Error(err))
	}
	appLog.Info("✅ Terhubung ke basis data pmb_app")

	sqlDB, _ := baseDB.DB()
	locker := lock.NewLocker(sqlDB)
	jobRunner := job.NewRunner(locker, appLog)

	// 2. Redis Client
	rdb, err := platformredis.ConnectRedis(cfg, appLog)
	if err != nil {
		appLog.Warn("Redis offline atau belum siap: " + err.Error())
	} else if rdb != nil {
		appLog.Info("✅ Terhubung ke Redis")
	}

	// 3. MinIO Storage
	minioClient, err := platformstorage.ConnectMinio(cfg, appLog)
	if err != nil {
		appLog.Warn("MinIO offline atau belum siap: " + err.Error())
	} else if minioClient != nil {
		appLog.Info("✅ Terhubung ke MinIO Storage")
	}

	// 4. Client Eksternal (Gateway & OCR)
	clients := client.NewClients(cfg)

	// 5. Inisialisasi Storage Client Wrapper
	var storageCli storage.Client
	if cfg.MinioEndpoint != "" {
		sCli, sErr := storage.NewClient(storage.Config{
			Endpoint:        cfg.MinioEndpoint,
			AccessKeyID:     cfg.MinioAccessKey,
			SecretAccessKey: cfg.MinioSecretKey,
			UseSSL:          cfg.MinioUseSSL,
		})
		if sErr == nil {
			storageCli = sCli
		}
	}

	// Inisialisasi Gin Engine
	r := gin.New()
	r.Use(pkglogger.GinRecovery(appLog))
	r.Use(requestid.Middleware())
	r.Use(pkglogger.GinLogger(appLog))

	// Health probes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "pmb-service"})
	})
	r.HEAD("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/health/ready", func(c *gin.Context) {
		dbStatus := "ok"
		if sqlDB == nil || sqlDB.Ping() != nil {
			dbStatus = "error"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ready",
			"database": dbStatus,
			"ocr":      cfg.OCREnabled,
		})
	})
	r.HEAD("/health/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// -------------------------------------------------------------------------
	// INISIALISASI 10 MODUL PMB
	// -------------------------------------------------------------------------

	// 1. Organization
	if _, err := organization.Init(r, baseDB, cfg.GatewaySharedSecret, cfg.InternalToken, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul organization", zap.Error(err))
	}

	// 2. Identity
	if _, err := identity.Init(r, baseDB, storageCli, clients.OCR, cfg.GatewaySharedSecret, cfg.InternalToken); err != nil {
		appLog.Fatal("Gagal inisialisasi modul identity", zap.Error(err))
	}

	// 3. Notification
	notifMod, err := notification.Init(r, baseDB, notification.Config{
		SMTPHost:      cfg.SMTPHost,
		SMTPPort:      cfg.SMTPPort,
		SMTPFromEmail: cfg.SMTPFromEmail,
		SMTPFromName:  cfg.SMTPFromName,
		InternalToken: cfg.InternalToken,
		GatewaySecret: cfg.GatewaySharedSecret,
	}, appLog)
	if err != nil {
		appLog.Fatal("Gagal inisialisasi modul notification", zap.Error(err))
	}

	// 4. Admission
	admissMod, err := admission.Init(r, baseDB, storageCli, admission.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		AuthURL:       cfg.GatewayInternalURL,
	}, appLog)
	if err != nil {
		appLog.Fatal("Gagal inisialisasi modul admission", zap.Error(err))
	}

	// 5. Finance
	if _, err := finance.Init(r, baseDB, storageCli, finance.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		IsDevelopment: cfg.AppEnv != "production",
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul finance", zap.Error(err))
	}

	// 5b. Referral (CR-014)
	if _, _, err := referral.Init(r, baseDB, referral.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul referral", zap.Error(err))
	}

	// 6. Exam
	if _, err := exam.Init(r, baseDB, storageCli, exam.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		IsDevelopment: cfg.AppEnv != "production",
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul exam", zap.Error(err))
	}

	// 7. Scoring
	if _, err := scoring.Init(r, baseDB, scoring.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		IsDevelopment: cfg.AppEnv != "production",
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul scoring", zap.Error(err))
	}

	// 8. Selection
	if _, err := selection.Init(r, baseDB, selection.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		IsDevelopment: cfg.AppEnv != "production",
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul selection", zap.Error(err))
	}

	// 9. Student
	studentMod, err := student.Init(r, baseDB, student.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
	}, appLog)
	if err != nil {
		appLog.Fatal("Gagal inisialisasi modul student", zap.Error(err))
	}

	// 10. NIM
	if _, err := nim.Init(r, baseDB, nim.Config{
		GatewaySecret: cfg.GatewaySharedSecret,
		InternalToken: cfg.InternalToken,
		AppPort:       cfg.AppPort,
		AuthURL:       cfg.GatewayInternalURL,
	}, appLog); err != nil {
		appLog.Fatal("Gagal inisialisasi modul nim", zap.Error(err))
	}

	// -------------------------------------------------------------------------
	// BACKGROUND JOBS
	// -------------------------------------------------------------------------
	if notifMod != nil && notifMod.Worker != nil {
		jobRunner.Register("notification_worker", 2*time.Second, func(ctx context.Context) error {
			notifMod.Worker.ProcessBatch(ctx)
			return nil
		})
	}

	if admissMod != nil && admissMod.OutboxWorkerCancel != nil {
		defer admissMod.OutboxWorkerCancel()
	}

	if studentMod != nil && studentMod.Service != nil {
		jobRunner.Register("student_outbox", 2*time.Second, func(ctx context.Context) error {
			_, err := studentMod.Service.ProcessPendingOutboxEvents(ctx)
			return err
		})
	}

	jobRunner.Start(context.Background())

	// -------------------------------------------------------------------------
	// SERVER HTTP
	// -------------------------------------------------------------------------
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		appLog.Info("PMB Service siap menerima request", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.Fatal("Gagal menjalankan server PMB Service", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLog.Info("Mematikan PMB Service secara anggun (graceful shutdown)...")

	jobRunner.Stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLog.Fatal("PMB Service dipaksa berhenti", zap.Error(err))
	}

	appLog.Info("PMB Service berhasil dihentikan")
}
