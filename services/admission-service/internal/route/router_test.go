package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
	"github.com/stretchr/testify/assert"
)

type dummyAdmissionSvc struct {
	service.AdmissionService
}

func (d *dummyAdmissionSvc) GetActivePeriod(ctx context.Context) (*model.Period, error) {
	return &model.Period{
		ID:           1,
		Name:         "Gelombang 1 2026/2027",
		AcademicYear: "2026/2027",
		IsActive:     true,
	}, nil
}

func (d *dummyAdmissionSvc) GetAdmissionPaths(ctx context.Context) ([]model.AdmissionPath, error) {
	return []model.AdmissionPath{
		{
			ID:              1,
			Name:            "Reguler",
			Code:            "REG",
			RegistrationFee: 250000,
			IsActive:        true,
		},
	}, nil
}

func (d *dummyAdmissionSvc) GetMyApplications(ctx context.Context, userID uuid.UUID) ([]model.Application, error) {
	return []model.Application{}, nil
}

func (d *dummyAdmissionSvc) GetMyActiveApplication(ctx context.Context, userID uuid.UUID) (*model.Application, error) {
	return nil, repository.ErrApplicationNotFound
}

func (d *dummyAdmissionSvc) CancelMyApplication(ctx context.Context, userID uuid.UUID, reason string) error {
	return nil
}

func (d *dummyAdmissionSvc) GetEligibility(ctx context.Context, userID uuid.UUID) (*service.EligibilityResponse, error) {
	return &service.EligibilityResponse{CanCreateNew: true}, nil
}

func (d *dummyAdmissionSvc) GetProgress(ctx context.Context, userID uuid.UUID) (*service.ProgressResponse, error) {
	return &service.ProgressResponse{CurrentStep: "step_choice"}, nil
}

func (d *dummyAdmissionSvc) ListApplicationsAdmin(ctx context.Context, filter repository.ApplicationFilter) ([]model.Application, int64, error) {
	return []model.Application{}, 0, nil
}

func setupTestAdmissionRouter(gatewaySecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := RouterConfig{
		GatewaySecret: gatewaySecret,
		InternalToken: "test-internal-token",
	}

	dummySvc := &dummyAdmissionSvc{}
	healthCtrl := controller.NewHealthController(nil)
	publicCtrl := controller.NewPublicController(dummySvc)
	appCtrl := controller.NewApplicationController(dummySvc)
	adminCtrl := controller.NewAdminController(dummySvc, nil, nil, nil, nil)
	internalCtrl := controller.NewInternalController(dummySvc)
	changeCtrl := controller.NewChangeController(nil)
	withdrawalCtrl := controller.NewWithdrawalController(nil)

	return SetupRouter(cfg, healthCtrl, publicCtrl, appCtrl, adminCtrl, internalCtrl, changeCtrl, withdrawalCtrl)
}

func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, role)
	if role == "admin_pmb" {
		req.Header.Set("X-User-Permissions", "admission.document.verify,dashboard.overview.read")
	}
}

func TestAdmissionRouter_PublicEndpoints(t *testing.T) {
	r := setupTestAdmissionRouter("test-secret-key")

	t.Run("GET /health returns 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /admission/periods/active returns 200 without auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/periods/active", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /admission/paths returns 200 without auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/paths", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAdmissionRouter_AuthenticatedEndpoints(t *testing.T) {
	secret := "test-secret-key"
	r := setupTestAdmissionRouter(secret)
	userID := uuid.New().String()

	t.Run("unauthenticated access to /admission/applications/me returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/applications/me", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("authenticated user can access /admission/applications/me", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/applications/me", nil)
		setGatewayAuthHeaders(req, secret, "req-1", userID, "user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("role user accessing /admission/admin/applications returns 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/applications", nil)
		setGatewayAuthHeaders(req, secret, "req-2", userID, "user")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("role admin_pmb accessing /admission/admin/applications returns 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/applications", nil)
		setGatewayAuthHeaders(req, secret, "req-3", userID, "admin_pmb")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
