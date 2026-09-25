package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
	IsDevelopment bool
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	progCtrl *controller.ProgramController,
	approvalCtrl *controller.ApprovalController,
	announcementCtrl *controller.AnnouncementController,
	studentCtrl *controller.StudentController,
	internalCtrl *controller.InternalController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health check publik
	r.GET("/health", healthCtrl.HealthCheck)
	r.GET("/health/ready", healthCtrl.ReadinessCheck)

	// 2. Internal endpoints (dilindungi X-Internal-Token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.GET("/results/:application_id", internalCtrl.GetResultByApplicationID)
	}

	// 3. Downstream endpoints dari API Gateway
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		registerSelectionRoutes := func(prefix string) {
			// Program Studi: Rekomendasi kelulusan
			programGroup := authGroup.Group(prefix + "/program")
			programGroup.Use(gatewayauth.RequireRole("prodi", "admin_pmb", "superadmin"))
			{
				programGroup.GET("/recommendations", progCtrl.GetRecommendation)
				programGroup.POST("/recommendations", progCtrl.SaveRecommendation)
				programGroup.PUT("/recommendations", progCtrl.SaveRecommendation)
				programGroup.POST("/recommendations/:id/submit", progCtrl.SubmitRecommendation)
			}

			// Pimpinan: Persetujuan / Pengembalian
			approvalGroup := authGroup.Group(prefix + "/approval")
			approvalGroup.Use(gatewayauth.RequireRole("pimpinan", "superadmin"))
			{
				approvalGroup.GET("/recommendations", approvalCtrl.ListApprovals)
				approvalGroup.POST("/recommendations/:id/approve", approvalCtrl.Approve)
				approvalGroup.POST("/recommendations/:id/return", approvalCtrl.Return)
			}

			// Admin PMB: Publikasi pengumuman
			adminGroup := authGroup.Group(prefix + "/admin")
			adminGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin"))
			{
				adminGroup.POST("/announcements", announcementCtrl.PublishAnnouncement)
			}

			// Calon Mahasiswa: Lihat hasil seleksi & tagihan daftar ulang
			meGroup := authGroup.Group(prefix + "/me")
			meGroup.Use(gatewayauth.RequireRole("calon_mahasiswa", "admin_pmb", "superadmin"))
			{
				meGroup.GET("/result", studentCtrl.GetMyResult)
			}
		}

		// Rute resmi dengan prefix /selection (sesuai API Gateway route rewrite)
		registerSelectionRoutes("/selection")
		// Alias tanpa prefix /selection untuk direct service call / pengujian langsung
		registerSelectionRoutes("")
	}

	return r
}
