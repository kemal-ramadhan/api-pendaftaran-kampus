package gatewayauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

const (
	HeaderGatewaySignature = "X-Gateway-Signature"
	HeaderGatewayTimestamp = "X-Gateway-Timestamp"
	HeaderRequestID        = "X-Request-ID"
	HeaderUserID           = "X-User-ID"
	DefaultMaxSkew         = 60 * time.Second
)

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
			response.Error(c, http.StatusForbidden, response.ErrCodeForbidden, "Akses ditolak: signature gateway tidak valid atau kedaluwarsa", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
