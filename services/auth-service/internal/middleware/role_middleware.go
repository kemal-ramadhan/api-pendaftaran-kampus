package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

// RequireRole memvalidasi apakah user memiliki salah satu dari role yang diizinkan
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(CtxUserRoles)
		if !exists {
			response.Unauthorized(c, response.ErrCodeTokenMissing, "Tidak terautentikasi")
			c.Abort()
			return
		}

		userRoles, ok := val.([]string)
		if !ok {
			response.Forbidden(c, response.ErrCodeForbidden, "Hak akses tidak valid")
			c.Abort()
			return
		}

		allowedMap := make(map[string]bool)
		for _, r := range allowedRoles {
			allowedMap[r] = true
		}

		hasRole := false
		for _, ur := range userRoles {
			if allowedMap[ur] {
				hasRole = true
				break
			}
		}

		if !hasRole {
			response.Error(c, http.StatusForbidden, response.ErrCodeForbidden, "Anda tidak memiliki izin untuk mengakses resource ini", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
