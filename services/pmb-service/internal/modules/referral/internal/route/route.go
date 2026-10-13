package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/controller"
)

func RegisterRoutes(r *gin.Engine, ctrl *controller.ReferralController) {
	// Register rute dengan prefix /api/v1/referral dan /referral
	groups := []*gin.RouterGroup{
		r.Group("/api/v1/referral"),
		r.Group("/referral"),
	}

	for _, g := range groups {
		// Public
		g.GET("/codes/:code/check", ctrl.CheckCode)

		// Camaba
		g.GET("/me", ctrl.GetMyReferral)
		g.PUT("/me", ctrl.AttachCode)
		g.DELETE("/me", ctrl.DeleteCode)

		// Partner (Guru BK)
		partner := g.Group("/partner")
		{
			partner.GET("/dashboard", ctrl.PartnerDashboard)
			partner.GET("/codes", ctrl.ListPartnerCodes)
			partner.GET("/codes/:id", ctrl.GetCodeDetail)
			partner.GET("/codes/:id/users", ctrl.ListCodeUsers)
			partner.GET("/codes/:id/users/export", ctrl.ExportCodeUsers)
			partner.POST("/transactions/approve", ctrl.ApproveTransactions)
			partner.POST("/transactions/:id/reject", ctrl.RejectTransaction)
			partner.GET("/codes/:id/settlements", ctrl.ListSettlements)
			partner.POST("/codes/:id/settlements", ctrl.SubmitSettlement)
			partner.GET("/codes/:id/commissions", ctrl.ListCommissions)
		}

		// Admin
		admin := g.Group("/admin")
		{
			admin.GET("/codes", ctrl.ListAdminCodes)
			admin.POST("/codes", ctrl.CreateCode)
			admin.POST("/pics/import", ctrl.ImportPICs)
			admin.GET("/settings", ctrl.GetSettings)
			admin.PUT("/settings", ctrl.UpdateSettings)
			admin.POST("/settlements/:id/verify", ctrl.VerifySettlement)
			admin.POST("/settlements/:id/reject", ctrl.RejectSettlement)
		}
	}
}
