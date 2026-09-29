package route_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/controller"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/route"
	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/service"
)

// Gunakan mock repository yang sama untuk router test
type mockFacultyRepo struct {
	faculties []model.Faculty
}

func (m *mockFacultyRepo) FindAll(ctx context.Context, isActive *bool) ([]model.Faculty, error) {
	return m.faculties, nil
}
func (m *mockFacultyRepo) FindByID(ctx context.Context, id uint) (*model.Faculty, error) {
	for _, f := range m.faculties {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, nil
}
func (m *mockFacultyRepo) FindByCode(ctx context.Context, code string) (*model.Faculty, error) {
	for _, f := range m.faculties {
		if f.Code == code {
			return &f, nil
		}
	}
	return nil, nil
}
func (m *mockFacultyRepo) Create(ctx context.Context, f *model.Faculty) error {
	f.ID = uint(len(m.faculties) + 1)
	m.faculties = append(m.faculties, *f)
	return nil
}
func (m *mockFacultyRepo) Update(ctx context.Context, f *model.Faculty) error { return nil }
func (m *mockFacultyRepo) Delete(ctx context.Context, id uint) error          { return nil }

type mockStudyProgramRepo struct {
	programs []model.StudyProgram
}

func (m *mockStudyProgramRepo) FindAll(ctx context.Context, facultyCode string, isActive *bool) ([]model.StudyProgram, error) {
	return m.programs, nil
}
func (m *mockStudyProgramRepo) FindByID(ctx context.Context, id uint) (*model.StudyProgram, error) {
	for _, p := range m.programs {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockStudyProgramRepo) FindByCode(ctx context.Context, code string) (*model.StudyProgram, error) {
	for _, p := range m.programs {
		if p.Code == code {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockStudyProgramRepo) FindByCodes(ctx context.Context, codes []string) ([]model.StudyProgram, error) {
	return m.programs, nil
}
func (m *mockStudyProgramRepo) Create(ctx context.Context, sp *model.StudyProgram) error {
	sp.ID = uint(len(m.programs) + 1)
	m.programs = append(m.programs, *sp)
	return nil
}
func (m *mockStudyProgramRepo) Update(ctx context.Context, sp *model.StudyProgram) error { return nil }
func (m *mockStudyProgramRepo) UpdateAdmissionOpen(ctx context.Context, code string, admissionOpen bool) error {
	return nil
}
func (m *mockStudyProgramRepo) Delete(ctx context.Context, id uint) error { return nil }

type mockInstitutionProfileRepo struct {
	profile *model.InstitutionProfile
}

func (m *mockInstitutionProfileRepo) Get(ctx context.Context) (*model.InstitutionProfile, error) {
	return m.profile, nil
}

func (m *mockInstitutionProfileRepo) Update(ctx context.Context, p *model.InstitutionProfile) error {
	m.profile = p
	return nil
}

const (
	testGatewaySecret = "test-secret"
	testInternalToken = "test-internal-token"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	f1 := model.Faculty{ID: 1, Code: "FT", Name: "Fakultas Teknik", IsActive: true}
	p1 := model.StudyProgram{ID: 1, FacultyID: 1, Faculty: &f1, Code: "TI", NIMCode: "55", Name: "Teknik Informatika", Degree: "S1", Accreditation: "Unggul", IsActive: true, AdmissionOpen: true}

	fRepo := &mockFacultyRepo{faculties: []model.Faculty{f1}}
	spRepo := &mockStudyProgramRepo{programs: []model.StudyProgram{p1}}
	profileRepo := &mockInstitutionProfileRepo{
		profile: &model.InstitutionProfile{
			FoundationName:  "Yayasan",
			InstitutionName: "UBTH",
			DocumentCode:    "UBTH",
		},
	}
	orgSvc := service.NewOrganizationService(fRepo, spRepo, profileRepo)

	pubCtl := controller.NewPublicController(orgSvc)
	admCtl := controller.NewAdminController(orgSvc)
	internalCtl := controller.NewInternalController(orgSvc)
	healthCtl := controller.NewHealthController(nil)

	route.SetupRouter(r, pubCtl, admCtl, internalCtl, healthCtl, testGatewaySecret, testInternalToken)
	return r
}

func addGatewayHeaders(req *http.Request, secret, userID, roles string) {
	reqID := "test-req-123"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := gatewayauth.GenerateSignature(secret, reqID, userID, ts)

	req.Header.Set(gatewayauth.HeaderRequestID, reqID)
	req.Header.Set(gatewayauth.HeaderGatewayTimestamp, ts)
	req.Header.Set(gatewayauth.HeaderGatewaySignature, sig)
	if userID != "" {
		req.Header.Set(gatewayauth.HeaderUserID, userID)
	}
	if roles != "" {
		req.Header.Set(gatewayauth.HeaderUserRoles, roles)
		if strings.Contains(roles, "superadmin") {
			req.Header.Set("X-User-Permissions", "organization.program.manage,organization.program.read")
		}
	}
}

func TestHealthEndpoints(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPublicStudyPrograms_WithGatewayAuth(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/organization/study-programs", nil)
	addGatewayHeaders(req, testGatewaySecret, "", "")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublicStudyPrograms_DirectBypass_Rejected(t *testing.T) {
	r := setupTestRouter()

	// Tanpa gateway signature headers
	req, _ := http.NewRequest(http.MethodGet, "/organization/study-programs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 unauthorized, got %d", w.Code)
	}
}

func TestAdminCreateStudyProgram_SuperadminAllowed(t *testing.T) {
	r := setupTestRouter()

	body, _ := json.Marshal(dto.CreateStudyProgramRequest{
		FacultyID:     1,
		Code:          "SI",
		NIMCode:       "57",
		Name:          "Sistem Informasi",
		Degree:        "S1",
		Accreditation: "Baik Sekali",
	})

	req, _ := http.NewRequest(http.MethodPost, "/organization/admin/study-programs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addGatewayHeaders(req, testGatewaySecret, "00000000-0000-0000-0000-000000000001", "superadmin")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminCreateStudyProgram_NonSuperadminForbidden(t *testing.T) {
	r := setupTestRouter()

	body, _ := json.Marshal(dto.CreateStudyProgramRequest{
		FacultyID:     1,
		Code:          "SI",
		NIMCode:       "57",
		Name:          "Sistem Informasi",
		Degree:        "S1",
		Accreditation: "Baik Sekali",
	})

	req, _ := http.NewRequest(http.MethodPost, "/organization/admin/study-programs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addGatewayHeaders(req, testGatewaySecret, "00000000-0000-0000-0000-000000000002", "calon_mahasiswa")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 forbidden, got %d", w.Code)
	}
}

func TestInternalGetStudyProgram_ValidToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/internal/study-programs/TI", nil)
	req.Header.Set("X-Internal-Token", testInternalToken)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInternalGetStudyProgram_InvalidToken(t *testing.T) {
	r := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/internal/study-programs/TI", nil)
	req.Header.Set("X-Internal-Token", "wrong-token")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 forbidden, got %d", w.Code)
	}
}
