package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/service"
	"github.com/stretchr/testify/assert"
)

func (d *dummyAdmissionSvc) GetHeaderSummary(ctx context.Context, periodID *uint, scope rbac.ScopeResult) (*service.HeaderSummaryResponse, error) {
	return &service.HeaderSummaryResponse{
		PeriodID:             1,
		PeriodName:           "Gelombang 1 2026/2027",
		TotalApplicants:      10,
		TotalApplicantsLabel: "10 Pendaftar Terdaftar",
		InstitutionName:      "Universitas Bakti Tunas Husada",
		CommitteeBadge:       "Panitia PMB",
	}, nil
}

func (d *dummyAdmissionSvc) GetOverview(ctx context.Context, periodID *uint, scope rbac.ScopeResult, role string) (*service.OverviewResponse, error) {
	tot := int64(10)
	return &service.OverviewResponse{
		PeriodID:   1,
		PeriodName: "Gelombang 1 2026/2027",
		KPIs: service.OverviewCardKPI{
			TotalApplicants: &tot,
		},
	}, nil
}

func (d *dummyAdmissionSvc) GetDocumentQueue(ctx context.Context, periodID *uint, status, q string, page, limit int, scope rbac.ScopeResult) (*service.DocumentQueueResponse, error) {
	res := &service.DocumentQueueResponse{
		Items: []service.DocumentQueueItem{},
	}
	return res, nil
}

func (d *dummyAdmissionSvc) GetApplicationDocumentsModal(ctx context.Context, appID uuid.UUID, scope rbac.ScopeResult) (*service.ApplicationDocumentsModalResponse, error) {
	return &service.ApplicationDocumentsModalResponse{
		ApplicationID: appID,
		TotalCount:    5,
		SummaryLabel:  "0/5 disetujui",
	}, nil
}

func (d *dummyAdmissionSvc) GetInstitutionDashboard(ctx context.Context, periodID *uint, level, code string, scope rbac.ScopeResult) (*service.InstitutionDashboardResponse, error) {
	return &service.InstitutionDashboardResponse{
		PeriodID:   1,
		Level:      level,
		TotalQuota: 100,
	}, nil
}

func (d *dummyAdmissionSvc) GetReportsPreview(ctx context.Context, periodID *uint, programCode string, scope rbac.ScopeResult) (*service.ReportPreviewResponse, error) {
	return &service.ReportPreviewResponse{
		Items: []service.ReportPreviewItem{},
	}, nil
}

func (d *dummyAdmissionSvc) GenerateOfficialReportPDF(ctx context.Context, docType string, periodID *uint, actorID uuid.UUID, actorName string) ([]byte, string, error) {
	return []byte("%PDF-1.4 mock official report %%EOF"), "BA-PMB-test.pdf", nil
}

func (d *dummyAdmissionSvc) GetCamabaDashboardV2(ctx context.Context, userID uuid.UUID) (*service.CamabaDashboardV2Response, error) {
	return &service.CamabaDashboardV2Response{
		Status:         "draft",
		StatusLabel:    "Menunggu Pembayaran",
		CompletedSteps: 1,
		TotalSteps:     6,
	}, nil
}

func (d *dummyAdmissionSvc) GetAdmissionOptions(ctx context.Context) (*service.AdmissionOptionsResponse, error) {
	return &service.AdmissionOptionsResponse{
		PeriodID:           1,
		PeriodName:         "Gelombang 1 2026/2027",
		ChoicesStatusLabel: "Pilihan Tersimpan",
		Paths:              []service.AdmissionPathOption{},
		StudyPrograms:      []service.StudyProgramOption{},
	}, nil
}

func TestAdminAdmission_RBACPermissions(t *testing.T) {
	secret := "test-secret-key"
	r := setupTestAdmissionRouter(secret)
	userID := uuid.New().String()

	t.Run("GET /admission/admin/header-summary - role with permission 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/header-summary", nil)
		setGatewayAuthHeaders(req, secret, "req-hs-1", userID, "admin_pmb")
		req.Header.Set("X-User-Permissions", "dashboard.overview.read")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /admission/admin/header-summary - role without permission 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/header-summary", nil)
		setGatewayAuthHeaders(req, secret, "req-hs-2", userID, "dosen")
		req.Header.Set("X-User-Permissions", "selection.interview.score")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("GET /admission/admin/document-queue - role without verify permission 403", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/document-queue", nil)
		setGatewayAuthHeaders(req, secret, "req-dq-1", userID, "keuangan")
		req.Header.Set("X-User-Permissions", "finance.reconciliation.read")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("GET /admission/admin/document-queue - role with verify permission 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/document-queue", nil)
		setGatewayAuthHeaders(req, secret, "req-dq-2", userID, "admin_pmb")
		req.Header.Set("X-User-Permissions", "admission.document.verify")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /admission/admin/reports/official.pdf - print official pdf 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/admin/reports/official.pdf?type=registrant", nil)
		setGatewayAuthHeaders(req, secret, "req-pdf-1", userID, "admin_pmb")
		req.Header.Set("X-User-Permissions", "report.official.print")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	})

	t.Run("GET /admission/options - public access 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/options", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET /admission/dashboard - camaba authenticated 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/admission/dashboard", nil)
		setGatewayAuthHeaders(req, secret, "req-dash", userID, "calon_mahasiswa")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
