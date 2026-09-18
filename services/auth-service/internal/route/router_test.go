package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	"go.uber.org/zap"
)

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AppEnv:        "test",
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	tm, err := token.NewTokenManager(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}

	authCtrl := controller.NewAuthController(nil, tm)
	healthCtrl := controller.NewHealthController(nil, nil, "test")

	r := gin.New()
	SetupRoutes(r, authCtrl, healthCtrl, tm, nil)

	// Uji route yang terdaftar
	endpoints := []struct {
		method   string
		path     string
		expected int
	}{
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodGet, "/health/ready", http.StatusServiceUnavailable},
		{http.MethodGet, "/auth/health", http.StatusServiceUnavailable},
		{http.MethodGet, "/.well-known/jwks.json", http.StatusOK},
		{http.MethodGet, "/auth/me", http.StatusUnauthorized}, // Tanpa token
	}

	for _, ep := range endpoints {
		req, _ := http.NewRequest(ep.method, ep.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != ep.expected {
			t.Errorf("%s %s status = %d; expected %d", ep.method, ep.path, w.Code, ep.expected)
		}
	}
}
