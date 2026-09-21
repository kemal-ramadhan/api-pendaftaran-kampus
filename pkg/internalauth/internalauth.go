package internalauth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

const HeaderInternalToken = "X-Internal-Token"

// Middleware memastikan request inter-service membawa X-Internal-Token yang sah
func Middleware(secretToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader(HeaderInternalToken))
		if secretToken == "" || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(secretToken)) != 1 {
			response.Error(c, http.StatusForbidden, response.ErrCodeForbidden, "Akses internal ditolak: token tidak valid", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
