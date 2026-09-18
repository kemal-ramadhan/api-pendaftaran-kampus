package route

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
)

func SetupRoutes(
	r *gin.Engine,
	authCtrl *controller.AuthController,
	healthCtrl *controller.HealthController,
	tokenMgr token.TokenManager,
	sessionStore token.SessionStore,
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
}
