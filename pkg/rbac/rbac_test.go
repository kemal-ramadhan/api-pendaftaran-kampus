package rbac_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequirePermission_Allow(t *testing.T) {
	r := gin.New()
	r.Use(rbac.RequirePermission("admission.document.verify", "dashboard.overview.read"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-Permissions", "dashboard.overview.read,other.perm")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequirePermission_Deny(t *testing.T) {
	r := gin.New()
	r.Use(rbac.RequirePermission("admission.document.verify"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-Permissions", "finance.reconciliation.read")

	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success   bool   `json:"success"`
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if resp.ErrorCode != rbac.ErrCodePermissionDenied {
		t.Fatalf("expected error_code PERMISSION_DENIED, got %s", resp.ErrorCode)
	}
}
