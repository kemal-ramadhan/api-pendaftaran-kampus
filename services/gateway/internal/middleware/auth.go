package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/gatewayconfig"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/jwks"
	"github.com/redis/go-redis/v9"
)

type ScopeClaim struct {
	Role string `json:"role"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

type GatewayClaims struct {
	Typ         string       `json:"typ"`
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	Roles       []string     `json:"roles"`
	Scopes      []ScopeClaim `json:"scopes"`
	Permissions []string     `json:"perms"`
	Sid         string       `json:"sid"`
	jwt.RegisteredClaims
}

const (
	CtxGatewayClaims = "gateway_claims"
)

// AuthAndRoleMiddleware memvalidasi token JWT via JWKS, denylist Redis, dan hak akses role
func AuthAndRoleMiddleware(jwksClient jwks.Client, rdb *redis.Client, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchedRoute, exists := GetMatchedRoute(c)
		if !exists {
			c.Next()
			return
		}

		if IsPublicRoute(c) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, response.ErrCodeTokenMissing, "Token autentikasi tidak disertakan")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Format token harus 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(parts[1])

		// Parse dan validasi JWT
		token, err := jwt.ParseWithClaims(tokenStr, &GatewayClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("metode signing token tidak terduga: %v", t.Header["alg"])
			}
			kidVal, ok := t.Header["kid"]
			if !ok {
				return nil, errors.New("header token tidak memiliki claim kid")
			}
			kid, ok := kidVal.(string)
			if !ok {
				return nil, errors.New("claim kid bukan string")
			}
			return jwksClient.GetKey(c.Request.Context(), kid)
		}, jwt.WithIssuer(cfg.JWTIssuer), jwt.WithAudience(cfg.JWTAudience))

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				response.Unauthorized(c, response.ErrCodeTokenExpired, "Token autentikasi telah kedaluwarsa")
				c.Abort()
				return
			}
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Token autentikasi tidak valid")
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*GatewayClaims)
		if !ok || !token.Valid {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Klaim token autentikasi tidak valid")
			c.Abort()
			return
		}

		// REGRESI CELAH NO. 7: Validasi tipe token typ=access
		if claims.Typ != "access" {
			response.Unauthorized(c, response.ErrCodeTokenInvalid, "Tipe token bukan access token")
			c.Abort()
			return
		}

		// Periksa denylist jti di Redis
		if rdb != nil && claims.ID != "" {
			denylistKey := fmt.Sprintf("auth:denylist:%s", claims.ID)
			exists, err := rdb.Exists(c.Request.Context(), denylistKey).Result()
			if err == nil && exists > 0 {
				response.Unauthorized(c, response.ErrCodeTokenRevoked, "Token autentikasi telah dicabut")
				c.Abort()
				return
			}
		}

		// Pemeriksaan Role jika route mensyaratkan role tertentu
		if len(matchedRoute.Roles) > 0 {
			userHasRole := false
			for _, requiredRole := range matchedRoute.Roles {
				for _, userRole := range claims.Roles {
					if userRole == requiredRole {
						userHasRole = true
						break
					}
				}
				if userHasRole {
					break
				}
			}

			if !userHasRole {
				response.Error(c, http.StatusForbidden, response.ErrCodeForbidden, "Anda tidak memiliki izin untuk mengakses resource ini", nil)
				c.Abort()
				return
			}
		}

		c.Set(CtxGatewayClaims, claims)
		c.Next()
	}
}

func GetGatewayClaims(c *gin.Context) (*GatewayClaims, bool) {
	val, exists := c.Get(CtxGatewayClaims)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*GatewayClaims)
	return claims, ok
}
