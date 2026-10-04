package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	publicCtrl *controller.PublicController,
	appCtrl *controller.ApplicationController,
	adminCtrl *controller.AdminController,
	internalCtrl *controller.InternalController,
	changeCtrl *controller.ChangeController,
	withdrawalCtrl *controller.WithdrawalController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health check endpoints
	r.GET("/health", healthCtrl.Health)
	r.HEAD("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)
	r.HEAD("/health/ready", healthCtrl.Readiness)

	// 2. Public admission info & verification
	publicGroup := r.Group("/admission")
	{
		publicGroup.GET("/periods/active", publicCtrl.GetActivePeriod)
		publicGroup.GET("/paths", publicCtrl.GetAdmissionPaths)
		publicGroup.GET("/options", publicCtrl.GetOptions)
		publicGroup.GET("/withdrawals/verify/:code", withdrawalCtrl.VerifyWithdrawalPublic)
	}
	r.GET("/withdrawals/verify/:code", withdrawalCtrl.VerifyWithdrawalPublic)

	// 3. Internal inter-service endpoints
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/applications/:id/status", internalCtrl.UpdateStatus)
		internalGroup.PUT("/applications/:id/status", internalCtrl.UpdateStatus)
		internalGroup.POST("/applications/:id/passed", internalCtrl.PassApplication)
		internalGroup.GET("/applications/:id", internalCtrl.GetApplication)
		internalGroup.GET("/applications/user/:user_id", internalCtrl.GetApplicationsByUserID)
		internalGroup.GET("/quotas", internalCtrl.GetQuota)
	}

	// 4. Gateway authenticated endpoints
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Camaba v2 Dashboard
		authGroup.GET("/admission/dashboard", publicCtrl.GetDashboard)

		// Pendaftar endpoints
		registerUserRoutes := func(prefix string) {
			userGroup := authGroup.Group(prefix)
			{
				userGroup.POST("", appCtrl.CreateApplication)
				userGroup.GET("/me", appCtrl.GetMyActiveApplication)
				userGroup.GET("/me/history", appCtrl.GetMyApplications)
				userGroup.GET("/me/eligibility", appCtrl.GetEligibility)
				userGroup.POST("/me/cancel", appCtrl.CancelApplication)
				userGroup.GET("/me/progress", appCtrl.GetProgress)
				userGroup.GET("/me/dashboard", publicCtrl.GetDashboard)
				userGroup.POST("/me/documents", appCtrl.UploadDocument)
				userGroup.DELETE("/me/documents/:code", appCtrl.DeleteDocument)
				userGroup.GET("/me/documents/:code/url", appCtrl.GetDocumentURL)
				userGroup.POST("/me/submit", appCtrl.SubmitApplication)

				// CR-006 Calon Mahasiswa (Riwayat & Pindah Jalur)
				userGroup.GET("/me/changes", changeCtrl.GetMyChanges)
				userGroup.GET("/me/path-change/options", changeCtrl.GetPathChangeOptions)
				userGroup.POST("/me/path-change", changeCtrl.RequestOrApplyPathChange)
				userGroup.DELETE("/me/path-change/pending", changeCtrl.CancelPendingPathChange)

				// CR-007 Calon Mahasiswa (Pengunduran Diri)
				userGroup.GET("/me/withdrawal/eligibility", withdrawalCtrl.GetEligibility)
				userGroup.POST("/me/withdrawal", withdrawalCtrl.RequestWithdrawal)
				userGroup.GET("/me/withdrawal", withdrawalCtrl.GetMyWithdrawal)
				userGroup.DELETE("/me/withdrawal/pending", withdrawalCtrl.CancelPendingWithdrawal)
				userGroup.GET("/me/withdrawal/letter", withdrawalCtrl.GetWithdrawalLetterPDF)

				// Menggunakan ID pendaftaran eksplisit
				userGroup.GET("/:id", appCtrl.GetApplicationDetail)
				userGroup.PUT("/:id/choices", appCtrl.ChangeChoices)
				userGroup.POST("/:id/documents", appCtrl.UploadDocument)
				userGroup.DELETE("/:id/documents/:code", appCtrl.DeleteDocument)
				userGroup.GET("/:id/documents/:code/url", appCtrl.GetDocumentURL)
				userGroup.POST("/:id/submit", appCtrl.SubmitApplication)
			}
		}

		registerUserRoutes("/admission/applications")
		registerUserRoutes("/applications")

		// Admin & Staf PMB endpoints (RBAC)
		registerAdminRoutes := func(prefix string) {
			adminGroup := authGroup.Group(prefix)
			{
				// Ringkasan & KPI (dashboard.overview.read)
				adminGroup.GET("/header-summary", rbac.RequirePermission("dashboard.overview.read"), adminCtrl.GetHeaderSummary)
				adminGroup.GET("/overview", rbac.RequirePermission("dashboard.overview.read"), adminCtrl.GetOverview)
				adminGroup.GET("/stats", rbac.RequirePermission("dashboard.overview.read"), adminCtrl.GetStats)

				// Verifikasi Berkas (admission.document.verify)
				adminGroup.GET("/document-queue", rbac.RequirePermission("admission.document.verify"), adminCtrl.GetDocumentQueue)
				adminGroup.GET("/applications/:id/documents", rbac.RequirePermission("admission.document.verify"), adminCtrl.GetApplicationDocuments)
				adminGroup.POST("/applications/:id/verify", rbac.RequirePermission("admission.document.verify"), adminCtrl.VerifyDocuments)
				adminGroup.PUT("/applications/:id/documents/:doc_id/verify", rbac.RequirePermission("admission.document.verify"), adminCtrl.VerifyDocuments)
				adminGroup.GET("/applications/:id/documents/:code/url", rbac.RequirePermission("admission.document.verify"), adminCtrl.GetDocumentURL)
				adminGroup.GET("/applications", rbac.RequirePermission("admission.document.verify", "dashboard.overview.read"), adminCtrl.ListApplications)
				adminGroup.GET("/applications/:id", rbac.RequirePermission("admission.document.verify", "dashboard.overview.read"), adminCtrl.GetApplicationDetail)

				// Dashboard Institusi (dashboard.institution.read)
				adminGroup.GET("/institution-dashboard", rbac.RequirePermission("dashboard.institution.read"), adminCtrl.GetInstitutionDashboard)

				// Laporan & Ekspor (report.export & report.official.print)
				adminGroup.GET("/reports/preview", rbac.RequirePermission("report.export"), adminCtrl.GetReportsPreview)
				adminGroup.GET("/reports/applicants/export", rbac.RequirePermission("report.export"), adminCtrl.ExportApplications)
				adminGroup.GET("/reports/export", rbac.RequirePermission("report.export"), adminCtrl.ExportApplications)
				adminGroup.GET("/applications/export", rbac.RequirePermission("report.export"), adminCtrl.ExportApplications)
				adminGroup.GET("/reports/official.pdf", rbac.RequirePermission("report.official.print"), adminCtrl.PrintOfficialReport)
				adminGroup.GET("/reports/:type/official.pdf", rbac.RequirePermission("report.official.print"), adminCtrl.PrintOfficialReport)

				// CR-006 Admin (Riwayat & Pindah Jalur - admission.path_change.manage)
				adminGroup.GET("/path-change-requests", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminListPathChangeRequests)
				adminGroup.GET("/path-changes/pending", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminListPathChangeRequests)
				adminGroup.POST("/path-change-requests/:id/approve", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminApprovePathChangeRequest)
				adminGroup.POST("/path-change-requests/:id/reject", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminRejectPathChangeRequest)
				adminGroup.POST("/applications/:id/move", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminMoveApplication)
				adminGroup.POST("/applications/:id/move-path", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminMoveApplication)
				adminGroup.POST("/path-changes/direct", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminMoveApplication)
				adminGroup.POST("/path-changes/:id/review", rbac.RequirePermission("admission.path_change.manage"), changeCtrl.AdminApprovePathChangeRequest)
				adminGroup.GET("/applications/:id/changes", rbac.RequirePermission("admission.path_change.manage", "admission.document.verify"), changeCtrl.AdminGetApplicationChanges)
				adminGroup.GET("/path-changes/export", rbac.RequirePermission("report.export", "admission.path_change.manage"), changeCtrl.AdminExportPathChanges)
				adminGroup.POST("/applications/:id/verify-docs", rbac.RequirePermission("admission.document.verify"), adminCtrl.VerifyDocuments)

				// CR-007 Admin (Pengunduran Diri - admission.withdrawal.manage)
				adminGroup.GET("/withdrawals", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminListWithdrawals)
				adminGroup.POST("/withdrawals/:id/approve", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminApproveWithdrawal)
				adminGroup.POST("/withdrawals/:id/reject", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminRejectWithdrawal)
				adminGroup.POST("/withdrawals/:id/review", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminApproveWithdrawal)
				adminGroup.POST("/applications/:id/withdraw", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminWithdrawApplication)
				adminGroup.POST("/withdrawals/direct", rbac.RequirePermission("admission.withdrawal.manage"), withdrawalCtrl.AdminWithdrawApplication)
				adminGroup.GET("/withdrawals/stats", rbac.RequirePermission("admission.withdrawal.manage", "dashboard.overview.read"), withdrawalCtrl.AdminGetWithdrawalStats)
				adminGroup.GET("/withdrawals/export", rbac.RequirePermission("report.export", "admission.withdrawal.manage"), withdrawalCtrl.AdminExportWithdrawalsXLSX)

				// Master data management (organization.program.manage)
				adminGroup.GET("/periods", adminCtrl.ListPeriods)
				adminGroup.POST("/periods", rbac.RequirePermission("organization.program.manage"), adminCtrl.CreatePeriod)
				adminGroup.GET("/paths", adminCtrl.ListPaths)
				adminGroup.POST("/paths", rbac.RequirePermission("organization.program.manage"), adminCtrl.CreatePath)
				adminGroup.GET("/quotas", adminCtrl.ListQuotas)
				adminGroup.POST("/quotas", rbac.RequirePermission("organization.program.manage"), adminCtrl.CreateQuota)
				adminGroup.GET("/requirements", adminCtrl.ListRequirements)
				adminGroup.POST("/requirements", rbac.RequirePermission("organization.program.manage"), adminCtrl.CreateRequirement)
			}
		}

		registerAdminRoutes("/admission/admin")
		registerAdminRoutes("/admin")
	}

	return r
}
