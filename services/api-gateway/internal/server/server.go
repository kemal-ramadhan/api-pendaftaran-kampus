package server

import (
	"github.com/gin-gonic/gin"
	pkglogger "github.com/pkm-asia/pmb-backend/pkg/logger"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/config"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/health"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/jwks"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/proxy"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SetupGatewayEngine mengonfigurasi Gin Engine dengan urutan middleware lengkap sesuai PRD Bagian 6.3
func SetupGatewayEngine(
	cfg *config.Config,
	jwksClient jwks.Client,
	rdb *redis.Client,
	proxyHandler *proxy.ProxyHandler,
	healthHandler *health.HealthHandler,
	appLog *zap.Logger,
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
	r.GET("/health/ready", healthHandler.ReadinessCheck)

	// 6. Blokir path terlarang (/internal/*, /metrics, /docs)
	r.Use(middleware.BlockProtectedPathsMiddleware())

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
