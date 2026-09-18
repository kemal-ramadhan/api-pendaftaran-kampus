package requestid

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextKey = "request_id"
	HeaderKey  = "X-Request-ID"
)

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

// Middleware memastikan setiap request memiliki Request ID valid.
// Jika header X-Request-ID sudah ada dan valid, gunakan nilai tersebut.
// Jika tidak ada atau format tidak valid, buat UUID v4 baru.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderKey)
		if reqID == "" || !validIDRegex.MatchString(reqID) {
			reqID = uuid.NewString()
		}

		c.Set(ContextKey, reqID)
		c.Writer.Header().Set(HeaderKey, reqID)
		c.Next()
	}
}

// Get mengambil Request ID dari gin.Context.
func Get(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if val, exists := c.Get(ContextKey); exists {
		if id, ok := val.(string); ok && id != "" {
			return id
		}
	}
	return c.GetHeader(HeaderKey)
}
