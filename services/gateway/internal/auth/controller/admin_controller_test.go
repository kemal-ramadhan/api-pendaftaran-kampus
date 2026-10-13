package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/dto"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/middleware"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/service"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func setupM2Router(t *testing.T) (*gin.Engine, token.TokenManager, *mockUserRepo, uuid.UUID, uuid.UUID) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AppEnv:        "test",
		BcryptCost:    bcrypt.MinCost,
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		InternalToken: "secret-internal-12345",
	}

	userRepo := &mockUserRepo{users: make(map[uuid.UUID]*model.User)}
	roleRepo := &mockRoleRepo{roles: map[string]*model.Role{
		model.RoleCalonMahasiswa: {ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"},
		model.RoleKeuangan:       {ID: 4, Code: model.RoleKeuangan, Name: "Keuangan"},
		model.RoleProdi:          {ID: 6, Code: model.RoleProdi, Name: "Prodi"},
		model.RoleSuperadmin:     {ID: 8, Code: model.RoleSuperadmin, Name: "Superadmin"},
	}}
	resetRepo := &mockResetRepo{}
	auditRepo := &mockAuditRepo{}
	mailer := &mockMailer{}
	sessionStore := &mockSessionStore{sessions: make(map[string]string)}

	logger := zap.NewNop()
	tm, err := token.NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}

	adminSvc := service.NewAdminService(
		cfg,
		userRepo,
		roleRepo,
		resetRepo,
		auditRepo,
		sessionStore,
		nil,
		mailer,
		nil,
		logger,
	)

	adminCtrl := NewAdminController(adminSvc)
	internalCtrl := NewInternalController(adminSvc)

	r := gin.New()
	auth := r.Group("/auth")
	admin := auth.Group("/admin")
	admin.Use(middleware.AuthMiddleware(tm, sessionStore))
	admin.Use(middleware.RequireRole(model.RoleSuperadmin))
	{
		admin.GET("/roles", adminCtrl.GetRoles)
		admin.GET("/users", adminCtrl.ListUsers)
		admin.POST("/users", adminCtrl.CreateUser)
		admin.GET("/users/:id", adminCtrl.GetUser)
		admin.PUT("/users/:id", adminCtrl.UpdateUser)
		admin.PUT("/users/:id/roles", adminCtrl.UpdateUserRoles)
		admin.PATCH("/users/:id/status", adminCtrl.UpdateUserStatus)
	}

	internal := r.Group("/internal")
	internal.Use(middleware.InternalAuthMiddleware(cfg.InternalToken))
	{
		internal.GET("/users/:id", internalCtrl.GetUser)
		internal.POST("/users/batch", internalCtrl.GetBatchUsers)
		internal.POST("/users/:id/roles", internalCtrl.AddUserRole)
	}

	// Seed Superadmin
	superadminID := uuid.New()
	emptyScope := ""
	superadmin := &model.User{
		ID:       superadminID,
		Email:    "superadmin@univ.ac.id",
		FullName: "Super Admin",
		Status:   model.UserStatusActive,
		UserRoles: []model.UserRole{
			{UserID: superadminID, RoleID: 8, Role: *roleRepo.roles[model.RoleSuperadmin], ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	}
	_ = userRepo.Create(context.Background(), superadmin)

	// Seed Calon Mahasiswa biasa
	calonID := uuid.New()
	calon := &model.User{
		ID:       calonID,
		Email:    "calon@example.com",
		FullName: "Calon Mahasiswa",
		Status:   model.UserStatusActive,
		UserRoles: []model.UserRole{
			{UserID: calonID, RoleID: 1, Role: *roleRepo.roles[model.RoleCalonMahasiswa], ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	}
	_ = userRepo.Create(context.Background(), calon)

	return r, tm, userRepo, superadminID, calonID
}

// Test RBAC Akses Admin: hanya superadmin yang boleh akses /auth/admin/*
func TestAdminController_RBAC(t *testing.T) {
	r, tm, _, superadminID, calonID := setupM2Router(t)

	// 1. Tanpa token -> 401 Unauthorized
	req1, _ := http.NewRequest(http.MethodGet, "/auth/admin/users", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("GET /auth/admin/users tanpa token harus 401; didapat %d", w1.Code)
	}

	// 2. Token role calon_mahasiswa -> 403 Forbidden
	tokenCalon, _, _, _ := tm.GenerateAccessToken(calonID, "calon@example.com", "Calon", []string{"calon_mahasiswa"}, nil, nil, uuid.NewString())
	req2, _ := http.NewRequest(http.MethodGet, "/auth/admin/users", nil)
	req2.Header.Set("Authorization", "Bearer "+tokenCalon)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("GET /auth/admin/users dengan token non-superadmin harus 403; didapat %d", w2.Code)
	}

	// 3. Token role superadmin -> 200 OK
	tokenSuper, _, _, _ := tm.GenerateAccessToken(superadminID, "superadmin@univ.ac.id", "Super Admin", []string{"superadmin"}, nil, nil, uuid.NewString())
	req3, _ := http.NewRequest(http.MethodGet, "/auth/admin/users", nil)
	req3.Header.Set("Authorization", "Bearer "+tokenSuper)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("GET /auth/admin/users dengan token superadmin harus 200; didapat %d", w3.Code)
	}
}

// Test Superadmin Membuat Akun Keuangan dan Prodi
func TestAdminController_CreateUser(t *testing.T) {
	r, tm, _, superadminID, _ := setupM2Router(t)
	tokenSuper, _, _, _ := tm.GenerateAccessToken(superadminID, "superadmin@univ.ac.id", "Super Admin", []string{"superadmin"}, nil, nil, uuid.NewString())

	// A. Buat User Keuangan
	payloadKeuangan := dto.AdminCreateUserRequest{
		Email:    "keuangan.staff@univ.ac.id",
		FullName: "Staff Keuangan",
		Phone:    "+628123456789",
		Password: "PasswordKeuangan1!",
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleKeuangan},
		},
	}
	bodyK, _ := json.Marshal(payloadKeuangan)
	reqK, _ := http.NewRequest(http.MethodPost, "/auth/admin/users", bytes.NewReader(bodyK))
	reqK.Header.Set("Authorization", "Bearer "+tokenSuper)
	reqK.Header.Set("Content-Type", "application/json")
	wK := httptest.NewRecorder()
	r.ServeHTTP(wK, reqK)

	if wK.Code != http.StatusCreated {
		t.Fatalf("POST /auth/admin/users create keuangan status = %d; expected 201. Body: %s", wK.Code, wK.Body.String())
	}

	// B. Buat User Prodi dengan Scope
	payloadProdi := dto.AdminCreateUserRequest{
		Email:    "prodi.ti@univ.ac.id",
		FullName: "Kaprodi Teknik Informatika",
		Phone:    "+628987654321",
		Password: "PasswordProdi1!",
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleProdi, ScopeType: "prodi", ScopeID: "TI"},
		},
	}
	bodyP, _ := json.Marshal(payloadProdi)
	reqP, _ := http.NewRequest(http.MethodPost, "/auth/admin/users", bytes.NewReader(bodyP))
	reqP.Header.Set("Authorization", "Bearer "+tokenSuper)
	reqP.Header.Set("Content-Type", "application/json")
	wP := httptest.NewRecorder()
	r.ServeHTTP(wP, reqP)

	if wP.Code != http.StatusCreated {
		t.Fatalf("POST /auth/admin/users create prodi status = %d; expected 201. Body: %s", wP.Code, wP.Body.String())
	}

	var res response.SuccessResponse
	_ = json.Unmarshal(wP.Body.Bytes(), &res)
	if !res.Success {
		t.Errorf("SuccessResponse.Success harus true")
	}
}

// Test Internal API dengan Header X-Internal-Token
func TestInternalController_AuthAndEndpoints(t *testing.T) {
	r, _, _, _, calonID := setupM2Router(t)

	// 1. Tanpa header X-Internal-Token -> 403 Forbidden
	req1, _ := http.NewRequest(http.MethodGet, "/internal/users/"+calonID.String(), nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusForbidden {
		t.Errorf("GET /internal/users/:id tanpa token harus 403; didapat %d", w1.Code)
	}

	// 2. Token internal salah -> 403 Forbidden
	req2, _ := http.NewRequest(http.MethodGet, "/internal/users/"+calonID.String(), nil)
	req2.Header.Set(middleware.HeaderInternalToken, "wrong-internal-token")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("GET /internal/users/:id token salah harus 403; didapat %d", w2.Code)
	}

	// 3. Token internal valid -> 200 OK
	req3, _ := http.NewRequest(http.MethodGet, "/internal/users/"+calonID.String(), nil)
	req3.Header.Set(middleware.HeaderInternalToken, "secret-internal-12345")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Errorf("GET /internal/users/:id token valid harus 200; didapat %d", w3.Code)
	}
}
