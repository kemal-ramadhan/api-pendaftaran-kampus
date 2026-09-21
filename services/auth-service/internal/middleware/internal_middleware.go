package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
)

const HeaderInternalToken = internalauth.HeaderInternalToken

// InternalAuthMiddleware memastikan request internal membawa token rahasia yang sah
func InternalAuthMiddleware(secretToken string) gin.HandlerFunc {
	return internalauth.Middleware(secretToken)
}
