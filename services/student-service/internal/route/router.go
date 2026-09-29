package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
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
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "pmb-student-service"})
	}
	readyHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "pmb-student-service"})
	}
	r.GET("/health", healthHandler)
	r.HEAD("/health", healthHandler)
	r.GET("/health/ready", readyHandler)
	r.HEAD("/health/ready", readyHandler)

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

			// Enrolment: Rekapitulasi Daftar Ulang & Kesiapan NIM
			pGroup.GET("/admin/enrolment",
				gatewayauth.RequireRole("superadmin", "admin_pmb", "keuangan", "prodi", "pimpinan"),
				rbac.RequirePermission("student.nim.issue"),
				studentCtrl.GetEnrolmentList,
			)

			// Penetapan NIM Manual & Batch
			pGroup.POST("/admin/students/:id/issue-nim",
				gatewayauth.RequireRole("superadmin", "admin_pmb"),
				rbac.RequirePermission("student.nim.issue"),
				studentCtrl.IssueNIMManual,
			)
			pGroup.POST("/admin/nim/batch",
				gatewayauth.RequireRole("superadmin", "admin_pmb"),
				rbac.RequirePermission("student.nim.issue"),
				studentCtrl.IssueNIMBatch,
			)

			// Gugus PKKMB
			pkkmbGroup := pGroup.Group("/admin/pkkmb")
			pkkmbGroup.Use(gatewayauth.RequireRole("superadmin", "admin_pmb"), rbac.RequirePermission("student.pkkmb.manage"))
			{
				pkkmbGroup.GET("/groups", studentCtrl.ListPKKMBGroups)
				pkkmbGroup.POST("/groups", studentCtrl.CreatePKKMBGroup)
				pkkmbGroup.PUT("/groups/:id", studentCtrl.UpdatePKKMBGroup)
				pkkmbGroup.DELETE("/groups/:id", studentCtrl.DeletePKKMBGroup)
				pkkmbGroup.POST("/auto-assign", studentCtrl.AutoAssignPKKMB)
				pkkmbGroup.PUT("/students/:student_id/group", studentCtrl.AssignPKKMBManual)
				pkkmbGroup.GET("/export", studentCtrl.ExportPKKMB)
			}

			// Ekspor SIAKAD
			pGroup.GET("/admin/siakad-export",
				gatewayauth.RequireRole("superadmin", "admin_pmb"),
				rbac.RequirePermission("student.activation.manage"),
				studentCtrl.ExportSIAKAD,
			)

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
