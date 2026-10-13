package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware mengatur header CORS terpusat berdasarkan whitelist origin
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowedMap := make(map[string]bool)
	for _, o := range allowedOrigins {
		clean := strings.TrimRight(strings.ToLower(strings.TrimSpace(o)), "/")
		if clean != "" {
			allowedMap[clean] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		cleanOrigin := strings.TrimRight(strings.ToLower(strings.TrimSpace(origin)), "/")

		if origin != "" && (allowedMap["*"] || allowedMap[cleanOrigin]) {
			// PERBAIKAN CELAH 10: Jika credentials true, jangan pakai wildcard '*'
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Internal-Token")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
