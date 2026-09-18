package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iamhexagoninc/go-jatidiri-app/internal/config"
)

type RateLimitOptions struct {
	Prefix    string
	Requests  int
	Window    time.Duration
	UseIPOnly bool
	KeyGetter func(*gin.Context) string
}

func RateLimiter(opts RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("RATE_LIMIT_ENABLED") == "false" {
			c.Next()
			return
		}

		// 🚀 PER-ROUTE UNLIMITED
		if opts.Requests <= 0 {
			c.Next()
			return
		}

		ip := clientIP(c.Request)
		if ip == "" {
			ip = "unknown"
		}

		key := ip
		if !opts.UseIPOnly && opts.KeyGetter != nil {
			if extra := opts.KeyGetter(c); extra != "" {
				key = fmt.Sprintf("%s:%s", ip, strings.ToLower(extra))
			}
		}

		redisKey := fmt.Sprintf("%s:%s", opts.Prefix, key)

		// Atomic INCR + EXPIRE via pipeline
		pipe := config.RDB.TxPipeline()
		cnt := pipe.Incr(config.Ctx(), redisKey)
		pipe.Expire(config.Ctx(), redisKey, opts.Window)
		_, err := pipe.Exec(config.Ctx())
		if err != nil {
			c.Next() // fallback kalau Redis error
			return
		}

		current := cnt.Val()
		if current > int64(opts.Requests) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(opts.Window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "too many requests, please slow down",
				"rate_limit":  opts.Requests,
				"window_secs": int(opts.Window.Seconds()),
			})
			c.Abort()
			return
		}

		ttl, _ := config.RDB.TTL(config.Ctx(), redisKey).Result()
		remaining := opts.Requests - int(current)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", opts.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", int(ttl.Seconds())))

		c.Next()
	}
}

func clientIP(r *http.Request) string {
	trustProxy := os.Getenv("TRUST_PROXY") == "true"

	if trustProxy {
		for _, h := range []string{"X-Forwarded-For", "X-Real-IP"} {
			if v := r.Header.Get(h); v != "" {
				parts := strings.Split(v, ",")
				return strings.TrimSpace(parts[0])
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
