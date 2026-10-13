package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/notification/internal/controller"
)

type RouterConfig struct {
	InternalToken       string
	GatewaySharedSecret string
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	notifCtrl *controller.NotificationController,
	actCtrl *controller.ActivityController,
	bcCtrl *controller.BroadcastController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, cfg, healthCtrl, notifCtrl, actCtrl, bcCtrl)
	return r
}

func RegisterRoutes(
	r *gin.Engine,
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	notifCtrl *controller.NotificationController,
	actCtrl *controller.ActivityController,
	bcCtrl *controller.BroadcastController,
) {
	// Health check publik untuk docker / load balancer
	r.GET("/health", healthCtrl.Health)
	r.HEAD("/health", healthCtrl.Health)
	r.GET("/health/ready", healthCtrl.Readiness)
	r.HEAD("/health/ready", healthCtrl.Readiness)

	// Admin endpoints (lewat Gateway dengan signature HMAC)
	adminGroup := r.Group("/notifications/admin")
	adminGroup.Use(gatewayauth.Middleware(cfg.GatewaySharedSecret))
	{
		// Log Aktivitas Panitia (PRD 2.1, 2.9)
		adminGroup.GET("/activity", rbac.RequirePermission("dashboard.overview.read"), actCtrl.GetRecentActivities)

		// Broadcast Notifikasi Massal (PRD 2.7)
		adminGroup.GET("/broadcast/templates", rbac.RequirePermission("broadcast.send"), bcCtrl.ListTemplates)
		adminGroup.POST("/broadcast/templates", rbac.RequirePermission("broadcast.send"), bcCtrl.CreateTemplate)
		adminGroup.PUT("/broadcast/templates/:id", rbac.RequirePermission("broadcast.send"), bcCtrl.UpdateTemplate)
		adminGroup.DELETE("/broadcast/templates/:id", rbac.RequirePermission("broadcast.send"), bcCtrl.DeleteTemplate)

		adminGroup.POST("/broadcast/preview", rbac.RequirePermission("broadcast.send"), bcCtrl.PreviewBroadcast)
		adminGroup.POST("/broadcast", rbac.RequirePermission("broadcast.send"), bcCtrl.SendBroadcast)
		adminGroup.GET("/broadcast", rbac.RequirePermission("broadcast.send"), bcCtrl.ListBroadcastLogs)
		adminGroup.GET("/broadcast/:id", rbac.RequirePermission("broadcast.send"), bcCtrl.GetBroadcastLog)
	}

	// Internal endpoints (dilindungi internal token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/notifications", notifCtrl.SendNotification)
		internalGroup.GET("/notifications/:id", notifCtrl.GetStatus)
		internalGroup.POST("/activity", actCtrl.RecordActivity)
	}
}
