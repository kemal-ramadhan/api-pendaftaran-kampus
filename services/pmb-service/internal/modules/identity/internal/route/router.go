package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	userCtrl *controller.UserController,
	adminCtrl *controller.AdminController,
	internalCtrl *controller.InternalController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, cfg, healthCtrl, userCtrl, adminCtrl, internalCtrl)
	return r
}

func RegisterRoutes(
	r *gin.Engine,
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	userCtrl *controller.UserController,
	adminCtrl *controller.AdminController,
	internalCtrl *controller.InternalController,
) {
	// 1. Health checks (tanpa auth)
	r.GET("/health", healthCtrl.Liveness)
	r.HEAD("/health", healthCtrl.Liveness)
	r.GET("/health/ready", healthCtrl.Readiness)
	r.HEAD("/health/ready", healthCtrl.Readiness)

	// 2. Internal inter-service endpoints
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.GET("/persons/:user_id", internalCtrl.GetPersonByID)
		internalGroup.GET("/persons/:user_id/completeness", internalCtrl.GetPersonCompleteness)
		internalGroup.POST("/persons/batch", internalCtrl.BatchGetPersons)
		internalGroup.POST("/persons/search-nik", internalCtrl.SearchNIK)
	}

	// 3. Downstream API endpoints (wajib signature gateway)
	gatewayGroup := r.Group("")
	gatewayGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Reference data endpoints (CR-015 44.2)
		gatewayGroup.GET("/identity/reference/:jenis", userCtrl.GetReferenceData)

		// Admin endpoints
		adminGroup := gatewayGroup.Group("/identity/admin")
		adminGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin"))
		{
			adminGroup.GET("/persons/:user_id", adminCtrl.GetPersonDetail)
		}

		// User personal endpoints
		meGroup := gatewayGroup.Group("/identity/me")
		{
			meGroup.GET("/profile", userCtrl.GetMyProfile)
			meGroup.PUT("/profile", userCtrl.UpdateMyProfile)
			meGroup.GET("/addresses", userCtrl.GetMyAddresses)
			meGroup.PUT("/addresses", userCtrl.SaveMyAddress)
			meGroup.GET("/guardians", userCtrl.GetMyGuardians)
			meGroup.PUT("/guardians", userCtrl.SaveMyGuardian)
			meGroup.GET("/education", userCtrl.GetMyEducation)
			meGroup.PUT("/education", userCtrl.SaveMyEducation)
			meGroup.PUT("/photo", userCtrl.UpdateMyPhoto)
			meGroup.GET("/completeness", userCtrl.GetMyCompleteness)

			// Integrasi KTP OCR
			meGroup.POST("/ktp-scan", userCtrl.InitiateKTPScan)
			meGroup.GET("/ktp-scan/:scan_id", userCtrl.GetKTPScanStatus)
			meGroup.POST("/ktp-scan/:scan_id/confirm", userCtrl.ConfirmKTPScan)
		}
	}
}
