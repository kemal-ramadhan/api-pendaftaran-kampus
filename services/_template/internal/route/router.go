package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/_template/internal/controller"
)

func SetupRouter(
	r *gin.Engine,
	ctl *controller.TemplateController,
	healthCtl *controller.HealthController,
	gatewaySecret string,
	internalToken string,
) {
	// Health routes (tanpa gatewayauth)
	r.GET("/health", healthCtl.LivenessCheck)
	r.HEAD("/health", healthCtl.LivenessCheck)
	r.GET("/health/ready", healthCtl.ReadinessCheck)
	r.HEAD("/health/ready", healthCtl.ReadinessCheck)

	// Protected downstream routes (Wajib gateway signature)
	api := r.Group("/template")
	api.Use(gatewayauth.Middleware(gatewaySecret))
	{
		api.GET("/items", ctl.ListItems)
		api.GET("/me", ctl.GetMyData)
	}

	// Internal inter-service routes (Wajib X-Internal-Token)
	internal := r.Group("/internal")
	internal.Use(internalauth.Middleware(internalToken))
	{
		internal.GET("/items", ctl.ListItems)
	}
}
