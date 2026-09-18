package route

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
)

func SetupRoutes(
	r *gin.Engine,
	authCtrl *controller.AuthController,
	adminCtrl *controller.AdminController,
	internalCtrl *controller.InternalController,
	healthCtrl *controller.HealthController,
	tokenMgr token.TokenManager,
	sessionStore token.SessionStore,
	internalToken string,
) {
	// Swagger Documentation
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health Check Endpoints
	r.GET("/health", healthCtrl.LivenessCheck)
	r.GET("/health/ready", healthCtrl.HealthCheck)
	r.GET("/auth/health", healthCtrl.HealthCheck)

	// Well-Known JWKS
	r.GET("/.well-known/jwks.json", authCtrl.GetJWKS)

	// Public Auth Endpoints
	auth := r.Group("/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/verify", authCtrl.VerifyOTP)
		auth.POST("/resend-otp", authCtrl.ResendOTP)
		auth.POST("/login", authCtrl.Login)
		auth.POST("/refresh", authCtrl.Refresh)
		auth.POST("/forgot-password", authCtrl.ForgotPassword)
		auth.POST("/reset-password", authCtrl.ResetPassword)
	}

	// Authenticated Auth Endpoints (All Roles)
	authProtected := auth.Group("")
	authProtected.Use(middleware.AuthMiddleware(tokenMgr, sessionStore))
	{
		authProtected.GET("/me", authCtrl.GetMe)
		authProtected.PUT("/profile", authCtrl.UpdateProfile)
		authProtected.PUT("/password", authCtrl.ChangePassword)
		authProtected.POST("/logout", authCtrl.Logout)
		authProtected.POST("/logout-all", authCtrl.LogoutAll)
	}

	// Admin Endpoints (Role: superadmin)
	admin := auth.Group("/admin")
	admin.Use(middleware.AuthMiddleware(tokenMgr, sessionStore))
	admin.Use(middleware.RequireRole(model.RoleSuperadmin))
	{
		admin.GET("/roles", adminCtrl.GetRoles)
		admin.GET("/users", adminCtrl.ListUsers)
		admin.POST("/users", adminCtrl.CreateUser)
		admin.GET("/users/:id", adminCtrl.GetUser)
		admin.PUT("/users/:id", adminCtrl.UpdateUser)
		admin.PUT("/users/:id/roles", adminCtrl.UpdateUserRoles)
		admin.PATCH("/users/:id/status", adminCtrl.UpdateUserStatus)
	}

	// Internal Endpoints (Inter-Service, tidak diekspos keluar)
	internal := r.Group("/internal")
	internal.Use(middleware.InternalAuthMiddleware(internalToken))
	{
		internal.GET("/users/:id", internalCtrl.GetUser)
		internal.POST("/users/batch", internalCtrl.GetBatchUsers)
		internal.POST("/users/:id/roles", internalCtrl.AddUserRole)
	}
}
