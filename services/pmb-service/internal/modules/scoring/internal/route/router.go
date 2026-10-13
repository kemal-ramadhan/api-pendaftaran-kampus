package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/scoring/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
	IsDevelopment bool
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	compCtrl *controller.ComponentController,
	examinerCtrl *controller.ExaminerController,
	progCtrl *controller.ProgramController,
	internalCtrl *controller.InternalController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	RegisterRoutes(r, cfg, healthCtrl, compCtrl, examinerCtrl, progCtrl, internalCtrl)
	return r
}

func RegisterRoutes(
	r *gin.Engine,
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	compCtrl *controller.ComponentController,
	examinerCtrl *controller.ExaminerController,
	progCtrl *controller.ProgramController,
	internalCtrl *controller.InternalController,
) {

	// 1. Health check publik
	r.GET("/health", healthCtrl.HealthCheck)
	r.HEAD("/health", healthCtrl.HealthCheck)
	r.GET("/health/ready", healthCtrl.ReadinessCheck)
	r.HEAD("/health/ready", healthCtrl.ReadinessCheck)

	// 2. Internal endpoints (dilindungi X-Internal-Token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.GET("/summaries", internalCtrl.GetSummaries)
		internalGroup.POST("/locks", internalCtrl.SetLock)
		internalGroup.GET("/locks", internalCtrl.GetLock)
	}

	// 3. Downstream endpoints dari API Gateway
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Master Komponen Nilai: Admin PMB & Superadmin
		adminGroup := authGroup.Group("/scoring/admin")
		adminGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin"))
		{
			adminGroup.POST("/components", compCtrl.CreateComponent)
			adminGroup.GET("/components", compCtrl.GetComponents)
			adminGroup.PUT("/components/:id", compCtrl.UpdateComponent)
			adminGroup.DELETE("/components/:id", compCtrl.DeleteComponent)
			adminGroup.POST("/paths/:path_id/validate-weight", compCtrl.ValidatePathWeight)
			adminGroup.PUT("/interview-scores/:application_id", rbac.RequirePermission("selection.interview.score"), examinerCtrl.InputInterviewScore)
		}

		// Input Nilai: Dosen Penguji, Admin PMB, Superadmin
		examinerGroup := authGroup.Group("/scoring/examiner")
		examinerGroup.Use(gatewayauth.RequireRole("dosen", "admin_pmb", "superadmin"))
		{
			examinerGroup.PUT("/scores/:application_id", examinerCtrl.InputScore)
			examinerGroup.GET("/scores/:application_id", examinerCtrl.GetParticipantScores)
		}

		// Rekapitulasi & Pemeringkatan: Prodi, Admin PMB, Superadmin, Pimpinan
		programGroup := authGroup.Group("/scoring/program")
		programGroup.Use(gatewayauth.RequireRole("prodi", "admin_pmb", "superadmin", "pimpinan"))
		{
			programGroup.GET("/recap", progCtrl.GetRecap)
			programGroup.GET("/recap/export", progCtrl.ExportRecap)
		}
	}
}
