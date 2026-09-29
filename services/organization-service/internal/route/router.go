package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
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
		org.GET("/institution-profile", pubCtl.GetInstitutionProfile)

		// Admin endpoints (RBAC permission)
		admin := org.Group("/admin")
		{
			admin.GET("/programs-overview", rbac.RequirePermission("organization.program.read", "organization.program.manage"), admCtl.GetProgramsOverview)
			admin.PATCH("/study-programs/:code/admission", rbac.RequirePermission("organization.program.manage"), admCtl.PatchAdmissionOpen)
			admin.GET("/institution-profile", rbac.RequirePermission("organization.program.read", "organization.program.manage"), admCtl.GetInstitutionProfile)
			admin.PUT("/institution-profile", rbac.RequirePermission("organization.program.manage"), admCtl.UpdateInstitutionProfile)

			admin.POST("/faculties", rbac.RequirePermission("organization.program.manage"), admCtl.CreateFaculty)
			admin.PUT("/faculties/:id", rbac.RequirePermission("organization.program.manage"), admCtl.UpdateFaculty)
			admin.PATCH("/faculties/:id", rbac.RequirePermission("organization.program.manage"), admCtl.PatchFaculty)

			admin.POST("/study-programs", rbac.RequirePermission("organization.program.manage"), admCtl.CreateStudyProgram)
			admin.PUT("/study-programs/:code", rbac.RequirePermission("organization.program.manage"), admCtl.UpdateStudyProgram)
			admin.PATCH("/study-programs/:code", rbac.RequirePermission("organization.program.manage"), admCtl.PatchStudyProgram)
		}
	}

	// Internal inter-service routes (Wajib X-Internal-Token)
	internal := r.Group("/internal")
	internal.Use(internalauth.Middleware(internalToken))
	{
		internal.GET("/study-programs", internalCtl.ListStudyPrograms)
		internal.GET("/study-programs/:code", internalCtl.GetStudyProgramByCode)
		internal.POST("/study-programs/batch", internalCtl.GetBatchStudyPrograms)
		internal.GET("/faculties/:code", internalCtl.GetFacultyByCode)
		internal.GET("/institution-profile", internalCtl.GetInstitutionProfile)
	}
}
