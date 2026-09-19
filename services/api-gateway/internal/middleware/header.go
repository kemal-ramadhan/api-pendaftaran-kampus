package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/requestid"
)

// HeaderPropagationMiddleware membersihkan header identitas palsu dari klien dan menginjeksi identitas terverifikasi
func HeaderPropagationMiddleware(sharedSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request

		// 1. Strip SELURUH header X-User-* dan X-Gateway-* dari klien (Anti-Spoofing!)
		for k := range req.Header {
			lower := strings.ToLower(k)
			if strings.HasPrefix(lower, "x-user-") || strings.HasPrefix(lower, "x-gateway-") {
				req.Header.Del(k)
			}
		}

		// 2. Pastikan X-Request-ID valid
		reqID := requestid.Get(c)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		req.Header.Set(gatewayauth.HeaderRequestID, reqID)

		// 3. Jika user terautentikasi, propagasikan klaim identitas
		userID := ""
		if claims, ok := GetGatewayClaims(c); ok {
			userID = claims.Subject
			req.Header.Set(gatewayauth.HeaderUserID, userID)
			req.Header.Set("X-User-Email", claims.Email)
			req.Header.Set("X-User-Roles", strings.Join(claims.Roles, ","))

			// Scopes di-encode JSON lalu base64url
			if len(claims.Scopes) > 0 {
				scopesJSON, err := json.Marshal(claims.Scopes)
				if err == nil {
					encodedScopes := base64.RawURLEncoding.EncodeToString(scopesJSON)
					req.Header.Set("X-User-Scopes", encodedScopes)
				}
			} else {
				req.Header.Set("X-User-Scopes", base64.RawURLEncoding.EncodeToString([]byte("[]")))
			}
		}

		// Injeksi Timestamp dan Signature HMAC untuk SEMUA request (publik maupun terautentikasi)
		nowStr := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set(gatewayauth.HeaderGatewayTimestamp, nowStr)

		if sharedSecret != "" {
			sig := gatewayauth.GenerateSignature(sharedSecret, reqID, userID, nowStr)
			req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
		}

		c.Next()
	}
}
