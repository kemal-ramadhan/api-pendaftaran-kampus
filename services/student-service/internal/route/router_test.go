package route

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
func (d *dummyStudentRepo) ListReregistrations(_ context.Context, _ string) ([]model.Reregistration, error) { return nil, nil }
func (d *dummyStudentRepo) CreateReregistration(_ context.Context, _ *model.Reregistration) error             { return nil }
func (d *dummyStudentRepo) UpdateReregistration(_ context.Context, _ *model.Reregistration) error             { return nil }
func (d *dummyStudentRepo) UpdateReregistrationInTx(_ *gorm.DB, _ *model.Reregistration) error               { return nil }
func (d *dummyStudentRepo) CreateStudentInTx(_ *gorm.DB, _ *model.Student) error                              { return nil }
func (d *dummyStudentRepo) GetStudentByID(_ context.Context, _ uuid.UUID) (*model.Student, error)            { return nil, nil }
func (d *dummyStudentRepo) GetStudentByUserID(_ context.Context, _ uuid.UUID) (*model.Student, error)         { return nil, nil }
func (d *dummyStudentRepo) GetStudentByApplicationID(_ context.Context, _ uuid.UUID) (*model.Student, error)  { return nil, nil }
func (d *dummyStudentRepo) ListStudents(_ context.Context, _ model.StudentFilter) ([]model.Student, int64, error) { return nil, 0, nil }
func (d *dummyStudentRepo) ListStudentsReadyForNIM(_ context.Context, _ string) ([]model.Student, error)      { return nil, nil }
func (d *dummyStudentRepo) ListStudentsWithNIM(_ context.Context, _ string) ([]model.Student, error)          { return nil, nil }
func (d *dummyStudentRepo) UpdateStudent(_ context.Context, _ *model.Student) error                          { return nil }
func (d *dummyStudentRepo) UpdateStudentNIMInTx(_ *gorm.DB, _ uuid.UUID, _ string) error                      { return nil }
func (d *dummyStudentRepo) MarkSiakadExported(_ context.Context, _ []uuid.UUID) error                         { return nil }

func (d *dummyStudentRepo) ListPKKMBGroups(_ context.Context, _ *uuid.UUID) ([]model.PKKMBGroup, error)      { return nil, nil }
func (d *dummyStudentRepo) GetPKKMBGroupByID(_ context.Context, _ uuid.UUID) (*model.PKKMBGroup, error)     { return nil, nil }
func (d *dummyStudentRepo) CreatePKKMBGroup(_ context.Context, _ *model.PKKMBGroup) error                     { return nil }
func (d *dummyStudentRepo) UpdatePKKMBGroup(_ context.Context, _ *model.PKKMBGroup) error                     { return nil }
func (d *dummyStudentRepo) DeletePKKMBGroup(_ context.Context, _ uuid.UUID) error                            { return nil }
func (d *dummyStudentRepo) GetPKKMBAssignmentByStudentID(_ context.Context, _ uuid.UUID) (*model.PKKMBAssignment, error) { return nil, nil }
func (d *dummyStudentRepo) CreateOrUpdatePKKMBAssignment(_ context.Context, _ *model.PKKMBAssignment) error  { return nil }
func (d *dummyStudentRepo) ListPKKMBAssignmentsByGroupID(_ context.Context, _ uuid.UUID) ([]model.PKKMBAssignment, error) { return nil, nil }
func (d *dummyStudentRepo) ListAllPKKMBAssignments(_ context.Context) ([]model.PKKMBAssignment, error)       { return nil, nil }
func (d *dummyStudentRepo) CountAssignedPerGroup(_ context.Context) (map[uuid.UUID]int, error)               { return make(map[uuid.UUID]int), nil }

func (d *dummyStudentRepo) CreateOutboxEventInTx(_ *gorm.DB, _ *model.OutboxEvent) error                      { return nil }
func (d *dummyStudentRepo) GetPendingOutboxEvents(_ context.Context, _ int) ([]model.OutboxEvent, error)       { return nil, nil }
func (d *dummyStudentRepo) UpdateOutboxEvent(_ context.Context, _ *model.OutboxEvent) error                     { return nil }

func setGatewayAuthHeaders(req *http.Request, secret, reqID, userID, role string, perms ...string) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, timestamp)

	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, timestamp)
	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderUserID, userID)
	req.Header.Set(gatewayauth.HeaderUserRoles, role)
	if len(perms) > 0 {
		req.Header.Set("X-User-Permissions", strings.Join(perms, ","))
	}
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
	svc := service.NewStudentService(repo, exp, nil, nil, nil, nil, nil, cfg)
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

	t.Run("GET /student/admin/enrolment dengan permission student.nim.issue diterima 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/student/admin/enrolment", nil)
		userID := uuid.New().String()
		reqID := uuid.New().String()

		setGatewayAuthHeaders(req, cfg.GatewaySecret, reqID, userID, "admin_pmb", "student.nim.issue")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /student/admin/pkkmb/groups dengan permission student.pkkmb.manage diterima 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/student/admin/pkkmb/groups", nil)
		userID := uuid.New().String()
		reqID := uuid.New().String()

		setGatewayAuthHeaders(req, cfg.GatewaySecret, reqID, userID, "admin_pmb", "student.pkkmb.manage")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
