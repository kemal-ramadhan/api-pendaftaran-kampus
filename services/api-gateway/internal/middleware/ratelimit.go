package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware membatasi laju request per IP secara global dan per route
func RateLimitMiddleware(rdb *redis.Client, defaultReq int, defaultWindow time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		limitReq := defaultReq
		limitWindow := defaultWindow
		routeKey := "global"

		if matched, ok := GetMatchedRoute(c); ok {
			routeKey = matched.Prefix
			if matched.RateLimit != nil {
				if matched.RateLimit.Requests > 0 {
					limitReq = matched.RateLimit.Requests
				}
				if matched.RateLimit.Window > 0 {
					limitWindow = matched.RateLimit.Window
				}
			}
		}

		key := fmt.Sprintf("gw:rl:%s:%s", routeKey, clientIP)

		pipe := rdb.TxPipeline()
		incr := pipe.Incr(c.Request.Context(), key)
		pipe.Expire(c.Request.Context(), key, limitWindow)
		_, err := pipe.Exec(c.Request.Context())

		if err == nil && incr.Val() > int64(limitReq) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeRateLimited, "Terlalu banyak permintaan, silakan coba beberapa saat lagi", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
