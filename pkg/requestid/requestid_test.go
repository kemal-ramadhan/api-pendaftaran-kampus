package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware_Generated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = Get(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	headerID := w.Header().Get(HeaderKey)
	if headerID == "" {
		t.Fatalf("expected %s header to be present", HeaderKey)
	}

	if capturedID != headerID {
		t.Fatalf("expected captured ID %s to equal header ID %s", capturedID, headerID)
	}
}

func TestRequestIDMiddleware_Existing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Middleware())

	customID := "custom-req-id-123"
	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = Get(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderKey, customID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedID != customID {
		t.Fatalf("expected custom ID %s, got %s", customID, capturedID)
	}

	if w.Header().Get(HeaderKey) != customID {
		t.Fatalf("expected response header %s, got %s", customID, w.Header().Get(HeaderKey))
	}
}
