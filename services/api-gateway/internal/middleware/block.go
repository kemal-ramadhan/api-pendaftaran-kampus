package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

// BlockProtectedPathsMiddleware memblokir akses dari publik ke endpoint internal hilir
func BlockProtectedPathsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.ToLower(c.Request.URL.Path)

		// Blokir path yang mengandung /internal, /metrics, atau /docs
		if strings.Contains(path, "/internal") || strings.Contains(path, "/metrics") || strings.Contains(path, "/docs") {
			response.Error(c, http.StatusNotFound, response.ErrCodeNotFound, "Resource tidak ditemukan", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
