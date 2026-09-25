package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
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
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health check endpoints
	r.GET("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)

	// 2. Public admission info
	publicGroup := r.Group("/admission")
	{
		publicGroup.GET("/periods/active", publicCtrl.GetActivePeriod)
		publicGroup.GET("/paths", publicCtrl.GetAdmissionPaths)
	}

	// 3. Internal inter-service endpoints
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/applications/:id/status", internalCtrl.UpdateStatus)
		internalGroup.PUT("/applications/:id/status", internalCtrl.UpdateStatus)
		internalGroup.GET("/applications/:id", internalCtrl.GetApplication)
		internalGroup.GET("/applications/user/:user_id", internalCtrl.GetApplicationsByUserID)
		internalGroup.GET("/quotas", internalCtrl.GetQuota)
	}

	// 4. Gateway authenticated endpoints
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
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
				userGroup.POST("/me/documents", appCtrl.UploadDocument)
				userGroup.DELETE("/me/documents/:code", appCtrl.DeleteDocument)
				userGroup.GET("/me/documents/:code/url", appCtrl.GetDocumentURL)
				userGroup.POST("/me/submit", appCtrl.SubmitApplication)

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

		// Admin, Staf Prodi, & Pimpinan endpoints
		adminGroup := authGroup.Group("/admission/admin")
		adminGroup.Use(gatewayauth.RequireRole("superadmin", "admin_pmb", "prodi", "pimpinan"))
		{
			adminGroup.GET("/stats", adminCtrl.GetStats)
			adminGroup.GET("/applications", adminCtrl.ListApplications)
			adminGroup.GET("/applications/export", adminCtrl.ExportApplications)
			adminGroup.GET("/applications/:id", adminCtrl.GetApplicationDetail)
			adminGroup.POST("/applications/:id/verify", adminCtrl.VerifyDocuments)
			adminGroup.GET("/applications/:id/documents/:code/url", adminCtrl.GetDocumentURL)

			// Master data management (PRD C.4.4 / TINGGI-1)
			adminGroup.GET("/periods", adminCtrl.ListPeriods)
			adminGroup.POST("/periods", gatewayauth.RequireRole("superadmin"), adminCtrl.CreatePeriod)
			adminGroup.GET("/paths", adminCtrl.ListPaths)
			adminGroup.POST("/paths", gatewayauth.RequireRole("superadmin"), adminCtrl.CreatePath)
			adminGroup.GET("/quotas", adminCtrl.ListQuotas)
			adminGroup.POST("/quotas", gatewayauth.RequireRole("superadmin"), adminCtrl.CreateQuota)
			adminGroup.GET("/requirements", adminCtrl.ListRequirements)
			adminGroup.POST("/requirements", gatewayauth.RequireRole("superadmin"), adminCtrl.CreateRequirement)
		}
	}

	return r
}
