package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("[NIM-SERVICE] Memulai inisialisasi PMB NIM Service...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[NIM-SERVICE] Gagal memuat konfigurasi: %v", err)
	}

	// 1. Inisialisasi Basis Data Postgres
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("[NIM-SERVICE] Gagal menghubungkan ke database Postgres: %v", err)
	}

	// Auto-migrate schema tabel
	if err := db.AutoMigrate(&model.NIMFormat{}, &model.NIMSequence{}, &model.NIMIssuance{}); err != nil {
		log.Fatalf("[NIM-SERVICE] Gagal melakukan AutoMigrate schema basis data: %v", err)
	}
	log.Println("[NIM-SERVICE] Schema basis data berhasil dimigrasi.")

	// Pastikan ada default format NIM
	var count int64
	db.Model(&model.NIMFormat{}).Count(&count)
	if count == 0 {
		defaultFormat := model.NIMFormat{
			ID:          uuid.New(),
			Pattern:     "{YY}{nim_code_prodi}{jalur_digit}{urut:4}",
			Description: "Format NIM Standar Universitas ({YY}=2 digit tahun, {nim_code_prodi}=2 digit prodi, {jalur_digit}=1 digit jalur, {urut:4}=4 digit urutan)",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_ = db.Create(&defaultFormat)
		log.Println("[NIM-SERVICE] Format default NIM berhasil diinisialisasi.")
	}

	// 2. Inisialisasi Inter-Service Clients
	studentCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.StudentServiceURL,
		InternalToken: cfg.InternalToken,
	})
	authCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.AuthServiceURL,
		InternalToken: cfg.InternalToken,
	})
	admissionCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.AdmissionServiceURL,
		InternalToken: cfg.InternalToken,
	})
	notificationCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.NotificationServiceURL,
		InternalToken: cfg.InternalToken,
	})
	organizationCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.OrganizationServiceURL,
		InternalToken: cfg.InternalToken,
	})

	// 3. Inisialisasi Lapisan Repositori & Domain Service
	nimRepo := repository.NewNIMRepository(db)
	nimSvc := service.NewNIMService(
		cfg,
		nimRepo,
		studentCli,
		authCli,
		admissionCli,
		notificationCli,
		organizationCli,
	)
	nimCtrl := controller.NewNIMController(nimSvc)

	// 4. Inisialisasi Router HTTP
	router := route.SetupRouter(cfg, nimCtrl)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("[NIM-SERVICE] Server aktif dan mendengarkan pada port %s (env: %s)\n", cfg.Port, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[NIM-SERVICE] Fatal server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[NIM-SERVICE] Menerima sinyal penghentian, mematikan server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[NIM-SERVICE] Server shutdown forced: %v", err)
	}

	fmt.Println("[NIM-SERVICE] Server berhasil dihentikan dengan aman.")
}
