package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/finance/internal/controller"
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
	RegisterRoutes(r, cfg, healthCtrl, invCtrl, payCtrl, rcCtrl, webhookCtrl, devCtrl, adminCtrl, internalCtrl, installmentCtrl, reportCtrl)
	return r
}

func RegisterRoutes(
	r *gin.Engine,
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
) {

	// 1. Health checks (publik)
	r.GET("/health", healthCtrl.Health)
	r.HEAD("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)
	r.HEAD("/health/ready", healthCtrl.Readiness)

	// 2. Webhook publik provider pembayaran & daftar metode pembayaran & config
	r.POST("/webhooks/payments/:provider", webhookCtrl.HandleWebhook)
	r.GET("/payment-methods", payCtrl.GetPaymentMethods)
	r.GET("/finance/payment-methods", payCtrl.GetPaymentMethods)
	r.GET("/payment-config", payCtrl.GetPaymentConfig)
	r.GET("/finance/payment-config", payCtrl.GetPaymentConfig)

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
		internalGroup.POST("/invoices/:id/cancel", internalCtrl.CancelInvoice)
		internalGroup.POST("/invoices/cancel-by-application/:application_id", internalCtrl.CancelInvoiceByApplication)
		internalGroup.POST("/invoices/:id/reprice", internalCtrl.RepriceInvoice)
		internalGroup.POST("/invoices/adjustment", internalCtrl.CreateAdjustmentInvoice)
		internalGroup.POST("/applications/:application_id/withdraw", internalCtrl.HandleApplicationWithdrawal)
	}

	// 5. Downstream endpoints dari API Gateway
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Tagihan calon mahasiswa
		registerCamabaInvoiceRoutes := func(base *gin.RouterGroup) {
			base.GET("", invCtrl.GetMyInvoices)
			base.GET("/:id", invCtrl.GetInvoiceDetail)
			base.POST("/:id/pay", payCtrl.PayWithVirtualAccount)
			base.POST("/:id/manual-transfer", payCtrl.PayWithManualTransfer)
			base.POST("/:id/renew", invCtrl.RenewInvoice)
			base.GET("/:id/receipt-view", invCtrl.GetReceiptView)
			base.GET("/:id/payments/:payment_id", payCtrl.GetPaymentDetail)
			base.GET("/:id/payments/:payment_id/status", payCtrl.GetPaymentStatus)
			base.POST("/:id/installment-request", installmentCtrl.RequestInstallment)
			base.GET("/:id/installments", installmentCtrl.GetInstallmentSchedule)
		}

		invGroup := authGroup.Group("/invoices/me")
		registerCamabaInvoiceRoutes(invGroup)

		invGroupAlt := authGroup.Group("/finance/me/invoices")
		registerCamabaInvoiceRoutes(invGroupAlt)

		// Kwitansi resmi
		receiptGroup := authGroup.Group("/receipts/me")
		{
			receiptGroup.GET("/:payment_id", rcCtrl.GetReceipt)
		}

		// Helper pendaftaran route admin (mendukung prefix /finance/admin maupun /admin)
		registerAdminRoutes := func(prefix string) {
			adminGroup := authGroup.Group(prefix)
			{
				// RBAC Terlindungi (docs/RBAC.md)
				// Rekonsiliasi & Ekspor
				adminGroup.GET("/reconciliation", rbac.RequirePermission("finance.reconciliation.read"), adminCtrl.GetReconciliation)
				adminGroup.GET("/reconciliation/export", rbac.RequirePermission("finance.reconciliation.read"), adminCtrl.ExportReconciliation)

				// Konfirmasi Manual
				adminGroup.POST("/invoices/:id/manual-confirm", rbac.RequirePermission("finance.payment.manual_confirm"), adminCtrl.ManualConfirmPayment)

				// Buku Besar (Ledger)
				adminGroup.GET("/ledger", rbac.RequirePermission("finance.ledger.read"), adminCtrl.GetLedger)
				adminGroup.GET("/ledger/export", rbac.RequirePermission("finance.ledger.read"), adminCtrl.ExportLedger)

				// Fee Schemes
				adminGroup.GET("/fee-schemes", rbac.RequirePermission("finance.fee.manage"), adminCtrl.ListFeeSchemes)
				adminGroup.POST("/fee-schemes", rbac.RequirePermission("finance.fee.manage"), adminCtrl.CreateFeeScheme)
				adminGroup.PUT("/fee-schemes/:id", rbac.RequirePermission("finance.fee.manage"), adminCtrl.CreateFeeScheme)

				// Pembayaran Manual & Verifikasi
				adminGroup.GET("/payments", rbac.RequirePermission("finance.payment.manual_confirm", "finance.reconciliation.read"), adminCtrl.ListManualTransfers)
				adminGroup.POST("/payments/:id/verify", rbac.RequirePermission("finance.payment.manual_confirm"), adminCtrl.VerifyPayment)

				// Cicilan (Installments)
				adminGroup.GET("/installment-requests", rbac.RequirePermission("finance.fee.manage", "finance.payment.manual_confirm"), installmentCtrl.ListInstallmentRequests)
				adminGroup.POST("/installment-requests/:id/review", rbac.RequirePermission("finance.fee.manage", "finance.payment.manual_confirm"), installmentCtrl.ReviewInstallmentRequest)

				// Refund Reviews
				adminGroup.GET("/refund-reviews", rbac.RequirePermission("finance.payment.manual_confirm"), adminCtrl.ListRefundReviews)
				adminGroup.PATCH("/refund-reviews/:id", rbac.RequirePermission("finance.payment.manual_confirm"), adminCtrl.UpdateRefundReview)

				// Rekonsiliasi Pembayaran otomatis
				adminGroup.POST("/reconcile", rbac.RequirePermission("finance.reconciliation.read"), adminCtrl.ReconcilePayments)

				// Cron Jatuh Tempo
				adminGroup.POST("/cron/process-due-dates", reportCtrl.ProcessDueDatesCron)

				// Laporan ringkasan & ekspor excel
				adminGroup.GET("/reports/summary", rbac.RequirePermission("finance.reconciliation.read", "finance.ledger.read"), reportCtrl.GetSummary)
				adminGroup.GET("/reports/export", rbac.RequirePermission("finance.reconciliation.read", "finance.ledger.read"), reportCtrl.ExportXLSX)
			}
		}

		registerAdminRoutes("/finance/admin")
		registerAdminRoutes("/admin")
	}
}
