package server

import (
	"github.com/gin-gonic/gin"
	pkglogger "github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	authcontroller "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/controller"
	authmiddleware "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/middleware"
	authmodel "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	authtoken "github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
	gatewayconfig "github.com/pkm-asia/pmb-backend/services/gateway/internal/gatewayconfig"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/health"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/jwks"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/proxy"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SetupGatewayEngine mengonfigurasi Gin Engine dengan urutan middleware lengkap sesuai PRD Bagian 6.3 dan Bagian 4
func SetupGatewayEngine(
	cfg *gatewayconfig.Config,
	jwksClient jwks.Client,
	rdb *redis.Client,
	proxyHandler *proxy.ProxyHandler,
	healthHandler *health.HealthHandler,
	authCtrl *authcontroller.AuthController,
	adminCtrl *authcontroller.AdminController,
	tokenMgr authtoken.TokenManager,
	sessionStore authtoken.SessionStore,
	appLog *zap.Logger,
	ssoControllers ...*authcontroller.SSOController,
) *gin.Engine {
	r := gin.New()

	// 1. Recovery
	r.Use(pkglogger.GinRecovery(appLog))

	// 2. Request ID
	r.Use(requestid.Middleware())

	// 3. Access Log
	r.Use(pkglogger.GinLogger(appLog))

	// 4. Security Headers
	r.Use(middleware.SecurityHeadersMiddleware())

	// 5. CORS (termasuk preflight OPTIONS 204)
	r.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	// Health probes langsung dari gateway
	r.GET("/health", healthHandler.LivenessCheck)
	r.HEAD("/health", healthHandler.LivenessCheck)
	r.GET("/health/ready", healthHandler.ReadinessCheck)
	r.HEAD("/health/ready", healthHandler.ReadinessCheck)

	// Well-known JWKS publik
	if authCtrl != nil {
		r.GET("/.well-known/jwks.json", authCtrl.GetJWKS)
		r.GET("/api/v1/auth/.well-known/jwks.json", authCtrl.GetJWKS)
	}

	// 6. Blokir path terlarang (/internal/*, /metrics, /docs)
	r.Use(middleware.BlockProtectedPathsMiddleware())

	// -------------------------------------------------------------------------
	// AUTH DIRECT IN-PROCESS ROUTING (PRD Bagian 4 butir 4)
	// -------------------------------------------------------------------------
	if authCtrl != nil && adminCtrl != nil {
		authAPI := r.Group("/api/v1/auth")
		{
			authAPI.GET("/health", healthHandler.LivenessCheck)
			authAPI.POST("/register", authCtrl.Register)
			authAPI.POST("/verify", authCtrl.VerifyOTP)
			authAPI.POST("/resend-otp", authCtrl.ResendOTP)
			authAPI.POST("/login", authCtrl.Login)
			authAPI.POST("/refresh", authCtrl.Refresh)
			authAPI.POST("/forgot-password", authCtrl.ForgotPassword)
			authAPI.POST("/reset-password", authCtrl.ResetPassword)

			// Authenticated user
			userGroup := authAPI.Group("")
			userGroup.Use(authmiddleware.AuthMiddleware(tokenMgr, sessionStore))
			{
				userGroup.GET("/me", authCtrl.GetMe)
				userGroup.GET("/me/permissions", authCtrl.GetMePermissions)
				userGroup.PUT("/profile", authCtrl.UpdateProfile)
				userGroup.PUT("/password", authCtrl.ChangePassword)
				userGroup.POST("/logout", authCtrl.Logout)
				userGroup.POST("/logout-all", authCtrl.LogoutAll)
			}

			// Admin user
			adminGroup := authAPI.Group("/admin")
			adminGroup.Use(authmiddleware.AuthMiddleware(tokenMgr, sessionStore))
			adminGroup.Use(authmiddleware.RequireRole(authmodel.RoleSuperadmin, authmodel.RoleAdminPMB))
			{
				adminGroup.GET("/roles", adminCtrl.GetRoles)
				adminGroup.GET("/permissions", adminCtrl.GetPermissions)
				adminGroup.GET("/roles/:code/permissions", adminCtrl.GetRolePermissions)
				adminGroup.PUT("/roles/:code/permissions", adminCtrl.UpdateRolePermissions)
				adminGroup.POST("/roles/:code/permissions", adminCtrl.UpdateRolePermissions)
				adminGroup.GET("/users", adminCtrl.ListUsers)
				adminGroup.POST("/users", adminCtrl.CreateUser)
				adminGroup.GET("/users/:id", adminCtrl.GetUser)
				adminGroup.PUT("/users/:id", adminCtrl.UpdateUser)
				adminGroup.PUT("/users/:id/roles", adminCtrl.UpdateUserRoles)
				adminGroup.PATCH("/users/:id/status", adminCtrl.UpdateUserStatus)
				adminGroup.GET("/service-accounts", adminCtrl.ListServiceAccounts)
				adminGroup.POST("/service-accounts", adminCtrl.CreateServiceAccount)
				adminGroup.GET("/service-accounts/:id", adminCtrl.GetServiceAccount)
				adminGroup.PUT("/service-accounts/:id", adminCtrl.UpdateServiceAccount)
				adminGroup.POST("/users/:id/service-accounts", adminCtrl.GrantUserAccountAccess)
				adminGroup.DELETE("/users/:id/service-accounts/:service_id", adminCtrl.RevokeUserAccountAccess)
			}
		}
	}

	// -------------------------------------------------------------------------
	// SSO / OPENID CONNECT ROUTING (CR-012)
	// -------------------------------------------------------------------------
	var ssoCtrl *authcontroller.SSOController
	if len(ssoControllers) > 0 {
		ssoCtrl = ssoControllers[0]
	}

	if ssoCtrl != nil {
		r.GET("/.well-known/openid-configuration", ssoCtrl.GetOpenIDConfiguration)
		r.GET("/api/v1/auth/.well-known/openid-configuration", ssoCtrl.GetOpenIDConfiguration)

		if cfg.SSOEnabled {
			oauth2 := r.Group("/oauth2")
			{
				oauth2.GET("/authorize", ssoCtrl.Authorize)
				oauth2.POST("/token", ssoCtrl.Token)
				oauth2.GET("/userinfo", ssoCtrl.UserInfo)
				oauth2.POST("/revoke", ssoCtrl.Revoke)
				oauth2.GET("/logout", ssoCtrl.Logout)
				oauth2.POST("/logout", ssoCtrl.Logout)
			}

			sso := r.Group("/sso")
			{
				sso.GET("/login", ssoCtrl.ShowLoginPage)
				sso.POST("/login", ssoCtrl.ProcessLogin)
			}
		}
	}

	// -------------------------------------------------------------------------
	// REVERSE PROXY UNTUK SEMUA ROUTE LAINNYA
	// -------------------------------------------------------------------------
	// 7. Route Matcher (longest prefix, public_paths matching)
	r.Use(middleware.MatchRouteMiddleware(cfg.Routes))

	// 8. Rate Limit (Global + Override per route berbasis Redis)
	r.Use(middleware.RateLimitMiddleware(rdb, cfg.RateLimitGlobalReq, cfg.RateLimitGlobalWin))

	// 9. Body Limit (default 1 MB atau override)
	r.Use(middleware.BodyLimitMiddleware())

	// 10. Auth (JWT RS256, JWKS, typ=access, Redis denylist) & Role Check
	r.Use(middleware.AuthAndRoleMiddleware(jwksClient, rdb, cfg))

	// 11. Strip client identity & Inject HMAC Gateway Signature
	r.Use(middleware.HeaderPropagationMiddleware(cfg.GatewaySharedSecret))

	// 12. Forward ke Reverse Proxy untuk semua route yang lolos
	r.NoRoute(func(c *gin.Context) {
		matched, ok := middleware.GetMatchedRoute(c)
		if !ok {
			response.NotFound(c, "Endpoint tidak ditemukan di API Gateway")
			return
		}
		proxyHandler.Forward(c, matched)
	})

	return r
}
