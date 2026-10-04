package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/controller"
)

func SetupRouter(
	cfg *config.Config,
	nimCtrl *controller.NIMController,
) *gin.Engine {
	if !cfg.IsDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health checks publik
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "pmb-nim-service"})
	}
	readyHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "pmb-nim-service"})
	}
	r.GET("/health", healthHandler)
	r.HEAD("/health", healthHandler)
	r.GET("/health/ready", readyHandler)
	r.HEAD("/health/ready", readyHandler)

	// 2. Endpoint Inter-Service Internal
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.POST("/nim/issue", nimCtrl.IssueNIM)
		internalGroup.GET("/nim/issuances/:application_id", nimCtrl.GetIssuanceByApplicationID)
		internalGroup.GET("/nim/by-application/:application_id", nimCtrl.GetIssuanceByApplicationID)
		internalGroup.POST("/nim/issuances/:application_id/void", nimCtrl.VoidNIM)
	}

	// 3. Endpoint Downstream API Gateway
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		registerNIMRoutes := func(prefix string) {
			pGroup := authGroup.Group(prefix)

			// Admin, Pimpinan, dan Program Studi: Riwayat Penerbitan NIM
			issuancesGroup := pGroup.Group("/admin/issuances")
			issuancesGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin", "prodi", "pimpinan"))
			{
				issuancesGroup.GET("", nimCtrl.ListIssuances)
			}

			// Format NIM Aktif (admin_pmb, superadmin) dengan permission student.nim.issue
			pGroup.GET("/admin/formats/active", gatewayauth.RequireRole("superadmin", "admin_pmb"), rbac.RequirePermission("student.nim.issue"), nimCtrl.ListActiveFormats)

			// Superadmin: Konfigurasi Format Template NIM
			formatsGroup := pGroup.Group("/admin/formats")
			formatsGroup.Use(gatewayauth.RequireRole("superadmin"), rbac.RequirePermission("student.nim.issue"))
			{
				formatsGroup.GET("", nimCtrl.ListFormats)
				formatsGroup.GET("/:id", nimCtrl.GetFormat)
				formatsGroup.POST("", nimCtrl.CreateFormat)
				formatsGroup.PUT("/:id", nimCtrl.UpdateFormat)
			}
		}

		registerNIMRoutes("/nim")
		registerNIMRoutes("")
	}

	return r
}
