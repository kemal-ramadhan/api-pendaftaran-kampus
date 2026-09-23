package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/student-service/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type dummyStudentRepo struct{}

func (d *dummyStudentRepo) DB() *gorm.DB                                                                     { return nil }
func (d *dummyStudentRepo) GetReregistrationByUserID(_ context.Context, _ uuid.UUID) (*model.Reregistration, error) { return nil, nil }
func (d *dummyStudentRepo) GetReregistrationByApplicationID(_ context.Context, _ uuid.UUID) (*model.Reregistration, error) { return nil, nil }
func (d *dummyStudentRepo) CreateReregistration(_ context.Context, _ *model.Reregistration) error             { return nil }
func (d *dummyStudentRepo) UpdateReregistration(_ context.Context, _ *model.Reregistration) error             { return nil }
func (d *dummyStudentRepo) UpdateReregistrationInTx(_ *gorm.DB, _ *model.Reregistration) error               { return nil }
func (d *dummyStudentRepo) CreateStudentInTx(_ *gorm.DB, _ *model.Student) error                              { return nil }
func (d *dummyStudentRepo) GetStudentByUserID(_ context.Context, _ uuid.UUID) (*model.Student, error)         { return nil, nil }
func (d *dummyStudentRepo) GetStudentByApplicationID(_ context.Context, _ uuid.UUID) (*model.Student, error)  { return nil, nil }
func (d *dummyStudentRepo) ListStudents(_ context.Context, _ model.StudentFilter) ([]model.Student, int64, error) { return nil, 0, nil }
func (d *dummyStudentRepo) UpdateStudentNIMInTx(_ *gorm.DB, _ uuid.UUID, _ string) error                      { return nil }
func (d *dummyStudentRepo) CreateOutboxEventInTx(_ *gorm.DB, _ *model.OutboxEvent) error                      { return nil }

func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, role)
}

func TestStudentRouter(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		Port:          "8091",
		IsDevelopment: true,
		GatewaySecret: "test-secret-gateway",
		InternalToken: "test-secret-internal",
	}

	repo := &dummyStudentRepo{}
	exp := exporter.NewStudentExporter()
	svc := service.NewStudentService(repo, exp, nil, nil, nil)
	ctrl := controller.NewStudentController(svc)
	r := SetupRouter(cfg, ctrl)

	t.Run("GET /health adalah publik dan mengembalikan status 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "pmb-student-service")
	})

	t.Run("GET /student/me/reregistration tanpa gateway signature ditolak 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/student/me/reregistration", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /student/me/reregistration dengan signature valid diterima gatewayauth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/student/me/reregistration", nil)
		userID := uuid.New().String()
		reqID := uuid.New().String()

		setGatewayAuthHeaders(req, cfg.GatewaySecret, reqID, userID, "calon_mahasiswa")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Bukan 401 unauthorized, melainkan masuk ke controller logic (422 NOT_PASSED karena user dummy belum lulus)
		assert.NotEqual(t, http.StatusUnauthorized, w.Code)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "NOT_PASSED")
	})
}
