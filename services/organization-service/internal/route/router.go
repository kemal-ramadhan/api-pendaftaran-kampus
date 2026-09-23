package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/controller"
)

func SetupRouter(
	r *gin.Engine,
	pubCtl *controller.PublicController,
	admCtl *controller.AdminController,
	internalCtl *controller.InternalController,
	healthCtl *controller.HealthController,
	gatewaySecret string,
	internalToken string,
) {
	// Health routes (tanpa gatewayauth)
	r.GET("/health", healthCtl.LivenessCheck)
	r.GET("/health/ready", healthCtl.ReadinessCheck)

	// Protected downstream routes (Wajib gateway signature)
	org := r.Group("/organization")
	org.Use(gatewayauth.Middleware(gatewaySecret))
	{
		// Public endpoints (digunakan pendaftaran calon mahasiswa)
		org.GET("/faculties", pubCtl.ListFaculties)
		org.GET("/study-programs", pubCtl.ListStudyPrograms)
		org.GET("/study-programs/:code", pubCtl.GetStudyProgramByCode)

		// Admin endpoints (Hanya role: superadmin)
		admin := org.Group("/admin")
		admin.Use(gatewayauth.RequireRole("superadmin"))
		{
			admin.POST("/faculties", admCtl.CreateFaculty)
			admin.PUT("/faculties/:id", admCtl.UpdateFaculty)
			admin.PATCH("/faculties/:id", admCtl.PatchFaculty)

			admin.POST("/study-programs", admCtl.CreateStudyProgram)
			admin.PUT("/study-programs/:id", admCtl.UpdateStudyProgram)
			admin.PATCH("/study-programs/:id", admCtl.PatchStudyProgram)
		}
	}

	// Internal inter-service routes (Wajib X-Internal-Token)
	internal := r.Group("/internal")
	internal.Use(internalauth.Middleware(internalToken))
	{
		internal.GET("/study-programs/:code", internalCtl.GetStudyProgramByCode)
		internal.POST("/study-programs/batch", internalCtl.GetBatchStudyPrograms)
	}
}
