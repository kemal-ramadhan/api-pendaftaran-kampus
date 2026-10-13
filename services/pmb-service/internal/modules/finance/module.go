package finance

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	platformdb "github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/db"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/config"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/provider"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/route"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/seeder"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Config struct {
	GatewaySecret string
	InternalToken string
	AppPort       string
	IsDevelopment bool
}

type Module struct {
	InvoiceService service.InvoiceService
	PaymentService service.PaymentService
}

func Init(r *gin.Engine, baseDB *gorm.DB, storageCli storage.Client, cfg Config, log *zap.Logger) (*Module, error) {
	db := platformdb.GetModuleDB(baseDB, "finance")
	_ = db.AutoMigrate(
		&model.FeeScheme{},
		&model.Invoice{},
		&model.Payment{},
		&model.Receipt{},
		&model.PaymentWebhookLog{},
		&model.InstallmentRequest{},
		&model.Installment{},
		&model.RefundReview{},
	)
	_ = seeder.SeedFeeSchemes(context.Background(), db)

	internalBaseURL := "http://127.0.0.1:" + cfg.AppPort
	newClient := func(url string) internalclient.Client {
		return internalclient.NewClient(internalclient.Config{
			BaseURL:       url,
			InternalToken: cfg.InternalToken,
			Timeout:       10 * time.Second,
		})
	}
	admissionCli := newClient(internalBaseURL)
	notifCli := newClient(internalBaseURL)

	paymentProvider := provider.NewMockProvider()
	reportExp := exporter.NewReportExporter()

	invRepo := repository.NewInvoiceRepository(db)
	payRepo := repository.NewPaymentRepository(db)
	rcRepo := repository.NewReceiptRepository(db)
	webhookLogRepo := repository.NewWebhookLogRepository(db)
	installmentRepo := repository.NewInstallmentRepository(db)
	reportRepo := repository.NewReportRepository(db)
	feeSchemeRepo := repository.NewFeeSchemeRepository(db)
	refundReviewRepo := repository.NewRefundReviewRepository(db)

	finCfg := &config.Config{
		AppEnv:              "development",
		AppPort:             8085,
		GatewaySharedSecret: cfg.GatewaySecret,
		InternalToken:       cfg.InternalToken,
	}

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
		finCfg,
	)
	paySvc.SetInstallmentRepo(installmentRepo)

	refundReviewSvc := service.NewRefundReviewService(refundReviewRepo)
	installmentSvc := service.NewInstallmentService(installmentRepo, invRepo, feeSchemeRepo, notifCli)
	reportSvc := service.NewReportService(reportRepo, reportExp)
	financeAdminSvc := service.NewFinanceAdminService(invRepo, payRepo, rcRepo, admissionCli, notifCli)

	healthCtrl := controller.NewHealthController(db)
	invCtrl := controller.NewInvoiceController(invSvc, financeAdminSvc)
	payCtrl := controller.NewPaymentController(paySvc, financeAdminSvc, finCfg)
	rcCtrl := controller.NewReceiptController(paySvc)
	webhookCtrl := controller.NewWebhookController(paySvc)
	devCtrl := controller.NewDevController(paySvc, finCfg)
	adminCtrl := controller.NewAdminController(paySvc, feeSchemeRepo, refundReviewSvc, financeAdminSvc)
	internalCtrl := controller.NewInternalController(invSvc)
	installmentCtrl := controller.NewInstallmentController(installmentSvc)
	reportCtrl := controller.NewReportController(reportSvc, installmentSvc)

	route.RegisterRoutes(r, route.RouterConfig{
		GatewaySecret: cfg.GatewaySecret,
		InternalToken: cfg.InternalToken,
		IsDevelopment: cfg.IsDevelopment,
	}, healthCtrl, invCtrl, payCtrl, rcCtrl, webhookCtrl, devCtrl, adminCtrl, internalCtrl, installmentCtrl, reportCtrl)

	return &Module{
		InvoiceService: invSvc,
		PaymentService: paySvc,
	}, nil
}
