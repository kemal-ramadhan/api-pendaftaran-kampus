package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

const HeaderInternalToken = "X-Internal-Token"

// InternalAuthMiddleware memastikan request internal membawa token rahasia yang sah
func InternalAuthMiddleware(secretToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(HeaderInternalToken)
		if secretToken == "" || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(secretToken)) != 1 {
			response.Error(c, http.StatusForbidden, response.ErrCodeForbidden, "Akses internal ditolak: token tidak valid", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
