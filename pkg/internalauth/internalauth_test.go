package internalauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInternalAuth_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "secret-internal-12345"

	r := gin.New()
	r.Use(Middleware(secret))
	r.GET("/internal/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "PONG")
	})

	// 1. Tanpa header token -> 403 Forbidden
	req1, _ := http.NewRequest(http.MethodGet, "/internal/ping", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w1.Code)
	}

	// 2. Token salah -> 403 Forbidden
	req2, _ := http.NewRequest(http.MethodGet, "/internal/ping", nil)
	req2.Header.Set(HeaderInternalToken, "wrong-token")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w2.Code)
	}

	// 3. Token benar -> 200 OK
	req3, _ := http.NewRequest(http.MethodGet, "/internal/ping", nil)
	req3.Header.Set(HeaderInternalToken, secret)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w3.Code)
	}
}
