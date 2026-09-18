package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const DefaultMaxBodyBytes = 1 * 1024 * 1024 // 1 MB default

func parseBytesSize(sizeStr string) int64 {
	clean := strings.ToUpper(strings.TrimSpace(sizeStr))
	if clean == "" {
		return DefaultMaxBodyBytes
	}
	if strings.HasSuffix(clean, "MB") {
		val, err := strconv.ParseInt(strings.TrimSuffix(clean, "MB"), 10, 64)
		if err == nil {
			return val * 1024 * 1024
		}
	}
	if strings.HasSuffix(clean, "KB") {
		val, err := strconv.ParseInt(strings.TrimSuffix(clean, "KB"), 10, 64)
		if err == nil {
			return val * 1024
		}
	}
	val, err := strconv.ParseInt(clean, 10, 64)
	if err == nil {
		return val
	}
	return DefaultMaxBodyBytes
}

// BodyLimitMiddleware membatasi ukuran payload body per route (default 1 MB)
func BodyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		maxBytes := int64(DefaultMaxBodyBytes)

		if matched, ok := GetMatchedRoute(c); ok && matched.MaxBody != "" {
			maxBytes = parseBytesSize(matched.MaxBody)
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
