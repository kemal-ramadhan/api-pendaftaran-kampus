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

	// 1. Request tanpa signature -> 401 GATEWAY_SIGNATURE_INVALID
	req1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("Request tanpa signature harus 401, didapat %d", w1.Code)
	}

	// 2. Request dengan signature salah / dimanipulasi -> 401 GATEWAY_SIGNATURE_INVALID
	reqID := "r-123"
	userID := "u-456"
	ts := fmt.Sprintf("%d", time.Now().Unix())
	req2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set(HeaderRequestID, reqID)
	req2.Header.Set(HeaderUserID, userID)
	req2.Header.Set(HeaderGatewayTimestamp, ts)
	req2.Header.Set(HeaderGatewaySignature, "invalidsignaturehash123")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Request dengan signature salah harus 401, didapat %d", w2.Code)
	}

	// 3. Request dengan timestamp kedaluwarsa (> 60s) -> 401 GATEWAY_SIGNATURE_INVALID
	oldTs := fmt.Sprintf("%d", time.Now().Add(-70*time.Second).Unix())
	oldSig := GenerateSignature(secret, reqID, userID, oldTs)
	req3, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set(HeaderRequestID, reqID)
	req3.Header.Set(HeaderUserID, userID)
	req3.Header.Set(HeaderGatewayTimestamp, oldTs)
	req3.Header.Set(HeaderGatewaySignature, oldSig)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusUnauthorized {
		t.Errorf("Request dengan timestamp kedaluwarsa harus 401, didapat %d", w3.Code)
	}

	// 4. Request dengan signature valid -> 200 OK
	validSig := GenerateSignature(secret, reqID, userID, ts)
	req4, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req4.Header.Set(HeaderRequestID, reqID)
	req4.Header.Set(HeaderUserID, userID)
	req4.Header.Set(HeaderGatewayTimestamp, ts)
	req4.Header.Set(HeaderGatewaySignature, validSig)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("Request dengan signature valid harus 200, didapat %d", w4.Code)
	}
}

func TestGatewayAuth_UserIdentityHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Extract valid identity and scopes", func(t *testing.T) {
		r := gin.New()
		r.GET("/identity", func(c *gin.Context) {
			uid, err := GetUserID(c)
			if err != nil {
				c.String(http.StatusBadRequest, err.Error())
				return
			}
			email := GetUserEmail(c)
			roles := GetUserRoles(c)
			scopes := GetUserScopes(c)
			scopeID, hasScope := GetScopeID(c, "prodi", "prodi")

			if uid.String() != "00000000-0000-0000-0000-000000000001" {
				c.String(http.StatusInternalServerError, "uid mismatch")
				return
			}
			if email != "user@kampus.ac.id" {
				c.String(http.StatusInternalServerError, "email mismatch")
				return
			}
			if len(roles) != 2 || !HasRole(c, "prodi") || !HasAnyRole(c, "admin", "prodi") {
				c.String(http.StatusInternalServerError, "role mismatch")
				return
			}
			if len(scopes) != 1 || !hasScope || scopeID != "TI" {
				c.String(http.StatusInternalServerError, "scope mismatch")
				return
			}
			c.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest(http.MethodGet, "/identity", nil)
		req.Header.Set(HeaderUserID, "00000000-0000-0000-0000-000000000001")
		req.Header.Set(HeaderUserEmail, "user@kampus.ac.id")
		req.Header.Set(HeaderUserRoles, "dosen, prodi")
		// [{"role":"prodi","type":"prodi","id":"TI"}] in base64url: W3sicm9sZSI6InByb2RpIiwidHlwZSI6InByb2RpIiwiaWQiOiJUSSJ9XQ
		req.Header.Set(HeaderUserScopes, "W3sicm9sZSI6InByb2RpIiwidHlwZSI6InByb2RpIiwiaWQiOiJUSSJ9XQ")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("RequireRole middleware", func(t *testing.T) {
		r := gin.New()
		r.GET("/admin-only", RequireRole("superadmin", "admin_pmb"), func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		// Unauthorized role
		req1, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
		req1.Header.Set(HeaderUserRoles, "calon_mahasiswa")
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		if w1.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w1.Code)
		}

		// Authorized role
		req2, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
		req2.Header.Set(HeaderUserRoles, "admin_pmb, dosen")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w2.Code)
		}
	})
}

