package route_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/selection-service/internal/route"
	"github.com/stretchr/testify/assert"
)

func TestRouterHealthAndAuth(t *testing.T) {
	healthCtrl := controller.NewHealthController()
	progCtrl := controller.NewProgramController(nil)
	approvalCtrl := controller.NewApprovalController(nil)
	announcementCtrl := controller.NewAnnouncementController(nil)
	studentCtrl := controller.NewStudentController(nil)
	internalCtrl := controller.NewInternalController(nil, nil)

	cfg := route.RouterConfig{
		GatewaySecret: "test-secret-at-least-32-bytes-long!",
		InternalToken: "secret-token",
		IsDevelopment: true,
	}

	r := route.SetupRouter(cfg, healthCtrl, progCtrl, approvalCtrl, announcementCtrl, studentCtrl, internalCtrl)

	// 1. Health check tanpa auth -> 200
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Internal endpoint tanpa token -> 403 Forbidden
	reqInternal, _ := http.NewRequest(http.MethodGet, "/internal/results/00000000-0000-0000-0000-000000000001", nil)
	wInternal := httptest.NewRecorder()
	r.ServeHTTP(wInternal, reqInternal)
	assert.Equal(t, http.StatusForbidden, wInternal.Code)

	// 3. Downstream tanpa gateway signature -> 401 Unauthorized
	reqProg, _ := http.NewRequest(http.MethodGet, "/selection/program/recommendations", nil)
	wProg := httptest.NewRecorder()
	r.ServeHTTP(wProg, reqProg)
	assert.Equal(t, http.StatusUnauthorized, wProg.Code)
}
