package requestid

import (
	"context"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextKey = "request_id"
	HeaderKey  = "X-Request-ID"
)

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)

type ctxKey struct{}

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
		if c.Request != nil {
			c.Request = c.Request.WithContext(NewContext(c.Request.Context(), reqID))
		}
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
	if c.Request != nil {
		if reqID := FromContext(c.Request.Context()); reqID != "" {
			return reqID
		}
	}
	return c.GetHeader(HeaderKey)
}

// NewContext memasukkan request ID ke standard context.Context
func NewContext(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, reqID)
}

// FromContext mengekstrak request ID dari context.Context (atau gin.Context)
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return Get(ginCtx)
	}
	if val := ctx.Value(ctxKey{}); val != nil {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	if val := ctx.Value(ContextKey); val != nil {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return ""
}
