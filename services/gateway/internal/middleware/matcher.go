package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/gatewayconfig"
)

const (
	CtxMatchedRoute = "matched_route"
	CtxIsPublic     = "is_public_route"
)

// MatchRouteMiddleware mencocokkan request path ke konfigurasi routes.yaml (longest prefix first)
func MatchRouteMiddleware(rc *config.RoutesConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		reqMethod := c.Request.Method

		// Endpoint health internal gateway dilewati dari proxy matching
		if reqPath == "/health" || reqPath == "/health/ready" {
			c.Next()
			return
		}

		var matched *config.RouteConfig
		for i := range rc.Routes {
			r := &rc.Routes[i]
			if strings.HasPrefix(reqPath, r.Prefix) {
				matched = r
				break
			}
		}

		if matched == nil {
			response.Error(c, http.StatusNotFound, response.ErrCodeNotFound, "Route API tidak ditemukan", nil)
			c.Abort()
			return
		}

		isPublic := matched.Auth == "public"

		if !isPublic && len(matched.PublicPaths) > 0 {
			// Hitung remainder setelah prefix
			remainder := strings.TrimPrefix(reqPath, matched.Prefix)
			if !strings.HasPrefix(remainder, "/") && remainder != "" {
				remainder = "/" + remainder
			}
			if remainder == "" {
				remainder = "/"
			}

			targetPattern := fmt.Sprintf("%s %s", reqMethod, remainder)
			targetPatternLower := strings.ToLower(targetPattern)
			for _, pub := range matched.PublicPaths {
				cleanPub := strings.TrimSpace(pub)
				cleanPubLower := strings.ToLower(cleanPub)
				if strings.EqualFold(cleanPub, targetPattern) ||
					strings.HasPrefix(targetPatternLower, cleanPubLower+"/") ||
					(strings.HasSuffix(cleanPubLower, "/*") && strings.HasPrefix(targetPatternLower, strings.TrimSuffix(cleanPubLower, "/*"))) {
					isPublic = true
					break
				}
			}
		}

		c.Set(CtxMatchedRoute, matched)
		c.Set(CtxIsPublic, isPublic)
		c.Next()
	}
}

func GetMatchedRoute(c *gin.Context) (*config.RouteConfig, bool) {
	val, exists := c.Get(CtxMatchedRoute)
	if !exists {
		return nil, false
	}
	r, ok := val.(*config.RouteConfig)
	return r, ok
}

func IsPublicRoute(c *gin.Context) bool {
	return c.GetBool(CtxIsPublic)
}
