package gatewayauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

const (
	HeaderGatewaySignature = "X-Gateway-Signature"
	HeaderGatewayTimestamp = "X-Gateway-Timestamp"
	HeaderRequestID        = "X-Request-ID"
	HeaderUserID           = "X-User-ID"
	HeaderUserEmail        = "X-User-Email"
	HeaderUserRoles        = "X-User-Roles"
	HeaderUserScopes       = "X-User-Scopes"
	HeaderUserPermissions  = "X-User-Permissions"
	DefaultMaxSkew         = 60 * time.Second
)

var (
	ErrUserIDMissing = errors.New("header X-User-ID tidak ditemukan")
	ErrUserIDInvalid = errors.New("format X-User-ID tidak valid")
)

// ScopeClaim merepresentasikan entitas scope hak akses pengguna
type ScopeClaim struct {
	Role string `json:"role"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

// GenerateSignature menghasilkan signature HMAC-SHA256 dari X-Request-ID + X-User-ID + timestamp
func GenerateSignature(secret, requestID, userID, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(requestID + userID + timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature memverifikasi integritas signature dan toleransi waktu timestamp (skew)
func VerifySignature(secret, requestID, userID, timestamp, signature string, maxSkew time.Duration) bool {
	if secret == "" || timestamp == "" || signature == "" {
		return false
	}

	unixSec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	ts := time.Unix(unixSec, 0)
	diff := time.Since(ts)
	if time.Duration(math.Abs(float64(diff))) > maxSkew {
		return false
	}

	expected := GenerateSignature(secret, requestID, userID, timestamp)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// Middleware membuat middleware Gin untuk memvalidasi bahwa request berasal dari API Gateway
func Middleware(sharedSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Jika sharedSecret tidak diset, abaikan verifikasi (hanya untuk development lokal tanpa gateway)
		if sharedSecret == "" {
			c.Next()
			return
		}

		sig := c.GetHeader(HeaderGatewaySignature)
		ts := c.GetHeader(HeaderGatewayTimestamp)
		reqID := c.GetHeader(HeaderRequestID)
		userID := c.GetHeader(HeaderUserID)

		if sig == "" || ts == "" || !VerifySignature(sharedSecret, reqID, userID, ts, sig, DefaultMaxSkew) {
			response.Error(c, http.StatusUnauthorized, response.ErrCodeGatewaySignatureInvalid, "Akses ditolak: signature gateway tidak valid atau kedaluwarsa", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID mengekstrak dan mem-parse X-User-ID dari header request
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.GetHeader(HeaderUserID))
	if raw == "" {
		return uuid.Nil, ErrUserIDMissing
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, ErrUserIDInvalid
	}
	return id, nil
}

// GetUserEmail mengambil X-User-Email dari header request
func GetUserEmail(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader(HeaderUserEmail))
}

// GetUserRoles mengambil daftar roles dari header X-User-Roles
func GetUserRoles(c *gin.Context) []string {
	raw := strings.TrimSpace(c.GetHeader(HeaderUserRoles))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	roles := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			roles = append(roles, trimmed)
		}
	}
	return roles
}

// HasRole memeriksa apakah pengguna memiliki salah satu dari roles yang ditentukan
func HasRole(c *gin.Context, requiredRole string) bool {
	for _, r := range GetUserRoles(c) {
		if r == requiredRole {
			return true
		}
	}
	return false
}

// HasAnyRole memeriksa apakah pengguna memiliki setidaknya salah satu dari daftar roles yang ditentukan
func HasAnyRole(c *gin.Context, requiredRoles ...string) bool {
	userRoles := GetUserRoles(c)
	for _, req := range requiredRoles {
		for _, ur := range userRoles {
			if ur == req {
				return true
			}
		}
	}
	return false
}

// GetUserScopes mem-parse header X-User-Scopes (base64url encoded JSON)
func GetUserScopes(c *gin.Context) []ScopeClaim {
	raw := strings.TrimSpace(c.GetHeader(HeaderUserScopes))
	if raw == "" {
		return nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		// Fallback ke standard base64 jika ada padding
		decoded, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil
		}
	}

	var scopes []ScopeClaim
	if err := json.Unmarshal(decoded, &scopes); err != nil {
		return nil
	}
	return scopes
}

// GetScopeID mencari scope ID untuk kombinasi role dan scopeType tertentu
func GetScopeID(c *gin.Context, role, scopeType string) (string, bool) {
	for _, s := range GetUserScopes(c) {
		if (role == "" || s.Role == role) && (scopeType == "" || s.Type == scopeType) {
			return s.ID, true
		}
	}
	return "", false
}

// RequireRole adalah middleware Gin untuk memastikan user memiliki salah satu dari roles yang ditentukan
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasAnyRole(c, roles...) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki hak akses untuk resource ini")
			c.Abort()
			return
		}
		c.Next()
	}
}
