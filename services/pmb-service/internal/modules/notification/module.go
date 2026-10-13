package notification

import (
	"time"

	"github.com/gin-gonic/gin"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/sender"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/service"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/template"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	SMTPHost      string
	SMTPPort      int
	SMTPFromEmail string
	SMTPFromName  string
	InternalToken string
	GatewaySecret string
}

type Module struct {
	NotificationService service.NotificationService
	Worker              *service.Worker
}

func Init(r *gin.Engine, baseDB *gorm.DB, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "notification")
	_ = db.AutoMigrate(&model.NotificationLog{}, &model.ActivityEvent{}, &model.BroadcastTemplate{}, &model.BroadcastLog{})

	var countTemplates int64
	db.Model(&model.BroadcastTemplate{}).Count(&countTemplates)
	if countTemplates == 0 {
		defaultTemplates := []model.BroadcastTemplate{
			{
				Code:         "PAYMENT_REMINDER",
				Name:         "Pengingat Batas Waktu Pembayaran",
				Channel:      "email",
				Subject:      "Pengingat Pembayaran Formulir PMB",
				BodyTemplate: "Halo {{NAMA_LENGKAP}}, tagihan formulir nomor registrasi {{NO_REGISTRASI}} sebesar {{TAGIHAN}} akan jatuh tempo pada {{BATAS_BAYAR}}. Mohon segera selesaikan pembayaran.",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			{
				Code:         "CBT_INSTRUCTION",
				Name:         "Instruksi Tahap Ujian CBT",
				Channel:      "email",
				Subject:      "Jadwal & Petunjuk Pelaksanaan Ujian CBT",
				BodyTemplate: "Halo {{NAMA_LENGKAP}}, ujian CBT pilihan prodi {{PRODI}} jalur {{JALUR}} dijadwalkan pada {{JADWAL_CBT}}. Siapkan kartu ujian dan koneksi internet stabil.",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		}
		_ = db.Create(&defaultTemplates).Error
	}

	notifCfg := &config.Config{
		AppEnv:                "development",
		InternalToken:         cfg.InternalToken,
		GatewaySharedSecret:   cfg.GatewaySecret,
		SMTPHost:              cfg.SMTPHost,
		SMTPPort:              cfg.SMTPPort,
		SMTPFromEmail:         cfg.SMTPFromEmail,
		SMTPFromName:          cfg.SMTPFromName,
		WADriver:              "log",
		WorkerIntervalSeconds: 2,
		WorkerBatchSize:       20,
	}

	emailSnd := sender.NewEmailSender(notifCfg)
	waSnd := sender.NewWhatsAppSender(notifCfg)
	tmplEngine := template.NewTemplateEngine()

	notifRepo := repository.NewNotificationRepository(db)
	notifSvc := service.NewNotificationService(notifRepo, emailSnd, waSnd, tmplEngine)
	actRepo := repository.NewActivityRepository(db)
	actSvc := service.NewActivityService(actRepo)
	bcRepo := repository.NewBroadcastRepository(db)
	bcSvc := service.NewBroadcastService(bcRepo)

	worker := service.NewWorker(notifSvc, 2*time.Second, 20)

	healthCtrl := controller.NewHealthController()
	notifCtrl := controller.NewNotificationController(notifSvc)
	actCtrl := controller.NewActivityController(actSvc)
	bcCtrl := controller.NewBroadcastController(bcSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		InternalToken:       cfg.InternalToken,
		GatewaySharedSecret: cfg.GatewaySecret,
	}, healthCtrl, notifCtrl, actCtrl, bcCtrl)

	return &Module{
		NotificationService: notifSvc,
		Worker:              worker,
	}, nil
}
