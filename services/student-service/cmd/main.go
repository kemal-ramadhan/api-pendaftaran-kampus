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

	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("[STUDENT-SERVICE] Memulai inisialisasi PMB Student Service...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[STUDENT-SERVICE] Gagal memuat konfigurasi: %v", err)
	}

	// 1. Inisialisasi Basis Data Postgres
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("[STUDENT-SERVICE] Gagal menghubungkan ke database Postgres: %v", err)
	}

	// Auto-migrate schema tabel
	if err := db.AutoMigrate(&model.Reregistration{}, &model.Student{}, &model.OutboxEvent{}); err != nil {
		log.Fatalf("[STUDENT-SERVICE] Gagal melakukan AutoMigrate schema basis data: %v", err)
	}
	log.Println("[STUDENT-SERVICE] Schema basis data berhasil dimigrasi.")

	// 2. Inisialisasi Inter-Service Clients
	admissionCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.AdmissionServiceURL,
		InternalToken: cfg.InternalToken,
	})
	selectionCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.SelectionServiceURL,
		InternalToken: cfg.InternalToken,
	})
	financeCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.FinanceServiceURL,
		InternalToken: cfg.InternalToken,
	})
	nimCli := internalclient.NewClient(internalclient.Config{
		BaseURL:       cfg.NIMServiceURL,
		InternalToken: cfg.InternalToken,
	})

	// 3. Inisialisasi Lapisan Repositori & Domain Service
	studentRepo := repository.NewStudentRepository(db)
	studentExporter := exporter.NewStudentExporter()

	studentSvc := service.NewStudentService(
		studentRepo,
		studentExporter,
		admissionCli,
		selectionCli,
		financeCli,
		nimCli,
	)

	studentCtrl := controller.NewStudentController(studentSvc)

	// 4. Inisialisasi Router HTTP
	router := route.SetupRouter(cfg, studentCtrl)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// 5. Background Outbox Worker (Kirim otomatis event nim_issuance_requested)
	outboxStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-outboxStop:
				return
			case <-ticker.C:
				_, _ = studentSvc.ProcessPendingOutboxEvents(context.Background())
			}
		}
	}()

	go func() {
		log.Printf("[STUDENT-SERVICE] Server aktif dan mendengarkan pada port %s (env: %s)\n", cfg.Port, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[STUDENT-SERVICE] Fatal server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	close(outboxStop)

	log.Println("[STUDENT-SERVICE] Menerima sinyal penghentian, mematikan server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[STUDENT-SERVICE] Server shutdown forced: %v", err)
	}

	fmt.Println("[STUDENT-SERVICE] Server berhasil dihentikan dengan aman.")
}
