package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
)

const (
	CtxUserIDKey   = "user_id"
	CtxUserEmail   = "user_email"
	CtxUserRoles   = "user_roles"
	CtxUserScopes  = "user_scopes"
	CtxSessionID   = "session_id"
	CtxTokenJTI    = "token_jti"
	CtxTokenExp    = "token_exp"
)

func AuthMiddleware(tm token.TokenManager, store token.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, response.ErrCodeTokenMissing, "Token autentikasi tidak disertakan")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Format token autentikasi harus 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := tm.ValidateAccessToken(tokenStr)
		if err != nil {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Token autentikasi tidak valid atau kedaluwarsa")
			c.Abort()
			return
		}

		// Cek denylist di Redis
		denylisted, err := store.IsDenylisted(c.Request.Context(), claims.ID)
		if err != nil || denylisted {
			response.Unauthorized(c, response.ErrCodeTokenRevoked, "Token autentikasi telah dicabut")
			c.Abort()
			return
		}

		userUUID, err := uuid.Parse(claims.Subject)
		if err != nil {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Identitas pengguna pada token tidak valid")
			c.Abort()
			return
		}

		var expTime time.Time
		if claims.ExpiresAt != nil {
			expTime = claims.ExpiresAt.Time
		}

		c.Set(CtxUserIDKey, userUUID)
		c.Set(CtxUserEmail, claims.Email)
		c.Set(CtxUserRoles, claims.Roles)
		c.Set(CtxUserScopes, claims.Scopes)
		c.Set(CtxSessionID, claims.Sid)
		c.Set(CtxTokenJTI, claims.ID)
		c.Set(CtxTokenExp, expTime)

		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(CtxUserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

func GetSessionID(c *gin.Context) string {
	return c.GetString(CtxSessionID)
}

func GetTokenJTI(c *gin.Context) string {
	return c.GetString(CtxTokenJTI)
}

func GetTokenExp(c *gin.Context) time.Time {
	if val, exists := c.Get(CtxTokenExp); exists {
		if t, ok := val.(time.Time); ok {
			return t
		}
	}
	return time.Now()
}
