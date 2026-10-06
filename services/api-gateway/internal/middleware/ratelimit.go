package middleware

import (
	"fmt"
	"net/http"
	"strings"
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

		// Naikkan batas di localhost/pengembangan lokal agar aktivitas dev/testing tidak terblokir
		isLocal := clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" ||
			strings.HasPrefix(clientIP, "127.") || strings.HasPrefix(clientIP, "::ffff:127.") ||
			strings.HasPrefix(clientIP, "192.168.") || strings.HasPrefix(clientIP, "10.") ||
			strings.HasPrefix(clientIP, "172.") || clientIP == ""
		if isLocal {
			if limitReq < 10000 {
				limitReq = 10000
			}
		}

		key := fmt.Sprintf("gw:rl:%s:%s", routeKey, clientIP)

		val, err := rdb.Incr(c.Request.Context(), key).Result()
		if err == nil {
			// Pasang expiration HANYA saat request pertama pada window tersebut (val == 1)
			if val == 1 {
				rdb.Expire(c.Request.Context(), key, limitWindow)
			} else {
				// Jaga-jaga jika kunci sebelumnya tidak memiliki TTL
				ttl, _ := rdb.TTL(c.Request.Context(), key).Result()
				if ttl <= 0 {
					rdb.Expire(c.Request.Context(), key, limitWindow)
				}
			}
		}

		if err == nil && val > int64(limitReq) {
			response.Error(c, http.StatusTooManyRequests, response.ErrCodeRateLimited, "Terlalu banyak permintaan, silakan coba beberapa saat lagi", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

