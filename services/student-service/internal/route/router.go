package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/controller"
)

func SetupRouter(
	cfg *config.Config,
	studentCtrl *controller.StudentController,
) *gin.Engine {
	if !cfg.IsDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health checks (publik)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "pmb-student-service"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "pmb-student-service"})
	})

	// 2. Internal inter-service endpoints
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.GET("/students/by-application/:application_id", studentCtrl.InternalGetStudentByApplicationID)
		internalGroup.PUT("/students/by-application/:application_id/nim", studentCtrl.InternalUpdateNIM)
		internalGroup.POST("/outbox/process", studentCtrl.ProcessOutbox)
		internalGroup.POST("/applications/:application_id/withdraw", studentCtrl.InternalWithdraw)
	}

	// 3. Downstream API Gateway endpoints
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		registerStudentRoutes := func(prefix string) {
			pGroup := authGroup.Group(prefix)

			// Calon Mahasiswa: Alur Daftar Ulang
			meGroup := pGroup.Group("/me/reregistration")
			meGroup.Use(gatewayauth.RequireRole("calon_mahasiswa", "admin_pmb", "superadmin"))
			{
				meGroup.GET("", studentCtrl.GetMyReregistration)
				meGroup.PUT("", studentCtrl.UpdateMyReregistration)
				meGroup.POST("/complete", studentCtrl.CompleteReregistration)
			}

			// Admin, Pimpinan, dan Program Studi: Rekap & Ekspor Mahasiswa Baru
			adminGroup := pGroup.Group("/admin/students")
			adminGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin", "prodi", "pimpinan"))
			{
				adminGroup.GET("", studentCtrl.ListStudents)
				adminGroup.GET("/export", studentCtrl.ExportStudentsXLSX)
			}
		}

		registerStudentRoutes("/student")
		registerStudentRoutes("")
	}

	return r
}
