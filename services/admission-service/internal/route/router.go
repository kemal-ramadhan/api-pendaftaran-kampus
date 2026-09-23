package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
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
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Health check endpoints
	r.GET("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)

	// Public admission info
	publicGroup := r.Group("/admission")
	{
		publicGroup.GET("/periods/active", publicCtrl.GetActivePeriod)
		publicGroup.GET("/paths", publicCtrl.GetAdmissionPaths)
	}

	// Gateway authenticated endpoints
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Pendaftar endpoints
		userGroup := authGroup.Group("/admission/applications")
		{
			userGroup.POST("", appCtrl.CreateApplication)
			userGroup.GET("/me", appCtrl.GetMyApplications)
			userGroup.GET("/:id", appCtrl.GetApplicationDetail)
			userGroup.PUT("/:id/choices", appCtrl.ChangeChoices)
		}

		// Admin & Staf Prodi endpoints
		adminGroup := authGroup.Group("/admission/admin")
		adminGroup.Use(gatewayauth.RequireRole("superadmin", "admin_pmb", "prodi"))
		{
			adminGroup.GET("/applications", adminCtrl.ListApplications)
			adminGroup.GET("/applications/:id", adminCtrl.GetApplicationDetail)
		}
	}

	return r
}
