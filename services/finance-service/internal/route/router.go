package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
	IsDevelopment bool
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	invCtrl *controller.InvoiceController,
	payCtrl *controller.PaymentController,
	rcCtrl *controller.ReceiptController,
	webhookCtrl *controller.WebhookController,
	devCtrl *controller.DevController,
	adminCtrl *controller.AdminController,
	internalCtrl *controller.InternalController,
	installmentCtrl *controller.InstallmentController,
	reportCtrl *controller.ReportController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health checks (publik)
	r.GET("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)

	// 2. Webhook publik provider pembayaran
	r.POST("/webhooks/payments/:provider", webhookCtrl.HandleWebhook)

	// 3. Endpoint simulasi development (hanya aktif pada development)
	if cfg.IsDevelopment {
		r.POST("/dev/payments/:id/simulate-paid", devCtrl.SimulatePaid)
	}

	// 4. Internal endpoints (dilindungi X-Internal-Token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/invoices", internalCtrl.CreateInvoice)
		internalGroup.GET("/invoices", internalCtrl.GetInvoices)
	}

	// 5. Downstream endpoints dari API Gateway
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Tagihan calon mahasiswa
		invGroup := authGroup.Group("/invoices/me")
		{
			invGroup.GET("", invCtrl.GetMyInvoices)
			invGroup.GET("/:id", invCtrl.GetInvoiceDetail)
			invGroup.POST("/:id/pay", payCtrl.PayWithVirtualAccount)
			invGroup.POST("/:id/manual-transfer", payCtrl.PayWithManualTransfer)
			invGroup.GET("/:id/payments/:payment_id", payCtrl.GetPaymentDetail)
			invGroup.POST("/:id/installment-request", installmentCtrl.RequestInstallment)
			invGroup.GET("/:id/installments", installmentCtrl.GetInstallmentSchedule)
		}

		// Kwitansi resmi
		receiptGroup := authGroup.Group("/receipts/me")
		{
			receiptGroup.GET("/:payment_id", rcCtrl.GetReceipt)
		}

		// Helper pendaftaran route admin (mendukung prefix /finance/admin maupun /admin)
		registerAdminRoutes := func(prefix string) {
			adminGroup := authGroup.Group(prefix)
			{
				// Verifikasi transfer manual & kelola cicilan: role keuangan & superadmin
				keuanganOnly := adminGroup.Group("")
				keuanganOnly.Use(gatewayauth.RequireRole("keuangan", "superadmin", "admin_pmb"))
				{
					keuanganOnly.GET("/payments", adminCtrl.ListManualTransfers)
					keuanganOnly.POST("/payments/:id/verify", adminCtrl.VerifyPayment)
					keuanganOnly.GET("/installment-requests", installmentCtrl.ListInstallmentRequests)
					keuanganOnly.POST("/installment-requests/:id/review", installmentCtrl.ReviewInstallmentRequest)
					keuanganOnly.POST("/cron/process-due-dates", reportCtrl.ProcessDueDatesCron)
					keuanganOnly.GET("/fee-schemes", adminCtrl.ListFeeSchemes)
					keuanganOnly.POST("/fee-schemes", adminCtrl.CreateFeeScheme)
				}

				// Laporan ringkasan & ekspor excel: role keuangan, pimpinan, superadmin
				reportGroup := adminGroup.Group("/reports")
				reportGroup.Use(gatewayauth.RequireRole("keuangan", "pimpinan", "superadmin", "admin_pmb"))
				{
					reportGroup.GET("/summary", reportCtrl.GetSummary)
					reportGroup.GET("/export", reportCtrl.ExportXLSX)
				}
			}
		}

		registerAdminRoutes("/finance/admin")
		registerAdminRoutes("/admin")
	}

	return r
}
