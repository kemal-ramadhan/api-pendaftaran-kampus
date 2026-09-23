package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/notification-service/internal/controller"
)

type RouterConfig struct {
	InternalToken string
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	notifCtrl *controller.NotificationController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Health check publik untuk docker / load balancer
	r.GET("/health", healthCtrl.Health)

	// Internal endpoints (dilindungi internal token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/notifications", notifCtrl.SendNotification)
		internalGroup.GET("/notifications/:id", notifCtrl.GetStatus)
	}

	return r
}
