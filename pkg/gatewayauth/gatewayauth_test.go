package gatewayauth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestGatewayAuth_VerifySignature(t *testing.T) {
	secret := "my-shared-secret-12345"
	reqID := "req-uuid-1"
	userID := "user-uuid-1"
	nowStr := strconv.FormatInt(time.Now().Unix(), 10)

	sig := GenerateSignature(secret, reqID, userID, nowStr)

	// Valid signature
	if !VerifySignature(secret, reqID, userID, nowStr, sig, 60*time.Second) {
		t.Fatal("Signature valid harus diterima")
	}

	// Tampered reqID
	if VerifySignature(secret, "tampered-req-id", userID, nowStr, sig, 60*time.Second) {
		t.Fatal("Signature dengan reqID yang diubah harus ditolak")
	}

	// Tampered userID
	if VerifySignature(secret, reqID, "tampered-user-id", nowStr, sig, 60*time.Second) {
		t.Fatal("Signature dengan userID yang diubah harus ditolak")
	}

	// Wrong secret
	if VerifySignature("wrong-secret", reqID, userID, nowStr, sig, 60*time.Second) {
		t.Fatal("Signature dengan secret salah harus ditolak")
	}

	// Skew > 60s
	pastStr := strconv.FormatInt(time.Now().Add(-70*time.Second).Unix(), 10)
	pastSig := GenerateSignature(secret, reqID, userID, pastStr)
	if VerifySignature(secret, reqID, userID, pastStr, pastSig, 60*time.Second) {
		t.Fatal("Signature dengan timestamp > 60 detik harus ditolak")
	}
}

func TestGatewayAuth_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "secret-gateway-shared-key"

	r := gin.New()
	r.Use(Middleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 1. Request tanpa headers -> 403
	req1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Errorf("Request tanpa signature harus 403, didapat %d", w1.Code)
	}

	// 2. Request dengan signature valid -> 200
	reqID := "r-123"
	userID := "u-456"
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := GenerateSignature(secret, reqID, userID, ts)

	req2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set(HeaderRequestID, reqID)
	req2.Header.Set(HeaderUserID, userID)
	req2.Header.Set(HeaderGatewayTimestamp, ts)
	req2.Header.Set(HeaderGatewaySignature, sig)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Request dengan signature valid harus 200, didapat %d", w2.Code)
	}
}
