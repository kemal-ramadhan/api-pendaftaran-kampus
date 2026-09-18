package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func setupTestAdminService(t *testing.T) (
	AdminService,
	*fakeUserRepo,
	*fakeRoleRepo,
	*fakeSessionStore,
	uuid.UUID, // superadmin ID
) {
	cfg := &config.Config{
		AppEnv:      "test",
		BcryptCost:  bcrypt.MinCost,
		FrontendURL: "http://localhost:5173",
	}

	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	resetRepo := newFakePasswordResetRepo()
	auditRepo := newFakeAuditRepo()
	sessionStore := newFakeSessionStore()
	mailer := newFakeMailer()
	logger := zap.NewNop()

	svc := NewAdminService(
		cfg,
		userRepo,
		roleRepo,
		resetRepo,
		auditRepo,
		sessionStore,
		mailer,
		logger,
	)

	// Seed 1 Superadmin awal
	superadminID := uuid.New()
	roleSuperadmin, _ := roleRepo.FindByCode(context.Background(), model.RoleSuperadmin)
	emptyScope := ""
	superadmin := &model.User{
		ID:       superadminID,
		Email:    "superadmin@example.com",
		FullName: "Super Administrator",
		Status:   model.UserStatusActive,
		UserRoles: []model.UserRole{
			{
				UserID:    superadminID,
				RoleID:    roleSuperadmin.ID,
				Role:      *roleSuperadmin,
				ScopeType: &emptyScope,
				ScopeID:   &emptyScope,
			},
		},
	}
	_ = userRepo.Create(context.Background(), superadmin)
	_ = roleRepo.AssignRole(context.Background(), &superadmin.UserRoles[0])

	return svc, userRepo, roleRepo, sessionStore, superadminID
}

// 1. Acceptance Criteria M2: Superadmin bisa membuat akun keuangan & prodi (dengan scope)
func TestAdminService_CreateUser_KeuanganAndProdi(t *testing.T) {
	svc, userRepo, _, _, superadminID := setupTestAdminService(t)
	ctx := context.Background()

	// A. Buat User Keuangan
	reqKeuangan := dto.AdminCreateUserRequest{
		Email:    "keuangan@univ.ac.id",
		FullName: "Staf Keuangan",
		Phone:    "081234567891",
		Password: "Password123!",
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleKeuangan},
		},
	}
	resKeuangan, err := svc.CreateUser(ctx, reqKeuangan, superadminID)
	if err != nil {
		t.Fatalf("CreateUser keuangan error: %v", err)
	}
	if resKeuangan.Email != "keuangan@univ.ac.id" {
		t.Errorf("Email keuangan = %s", resKeuangan.Email)
	}
	if len(resKeuangan.Roles) != 1 || resKeuangan.Roles[0] != model.RoleKeuangan {
		t.Errorf("Roles keuangan = %v; expected [keuangan]", resKeuangan.Roles)
	}
	if resKeuangan.Status != model.UserStatusActive {
		t.Errorf("Status user baru via admin harus langsung active, didapat: %s", resKeuangan.Status)
	}

	// B. Buat User Prodi dengan Scope
	reqProdi := dto.AdminCreateUserRequest{
		Email:    "kaprodi.ti@univ.ac.id",
		FullName: "Kaprodi TI",
		Phone:    "081234567892",
		Password: "Password123!",
		Roles: []dto.RoleAssignmentDTO{
			{
				Code:      model.RoleProdi,
				ScopeType: "prodi",
				ScopeID:   "TI",
			},
		},
	}
	resProdi, err := svc.CreateUser(ctx, reqProdi, superadminID)
	if err != nil {
		t.Fatalf("CreateUser prodi error: %v", err)
	}
	if len(resProdi.Scopes) != 1 || resProdi.Scopes[0].ID != "TI" {
		t.Errorf("Scopes prodi = %v; expected scope ID 'TI'", resProdi.Scopes)
	}

	// C. Validasi penolakan scope prodi yang tidak valid
	reqProdiInvalidScope := dto.AdminCreateUserRequest{
		Email:    "invalid.prodi@univ.ac.id",
		FullName: "Prodi Invalid",
		Phone:    "081234567893",
		Password: "Password123!",
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleProdi, ScopeID: "x"}, // Kurang dari 2 karakter
		},
	}
	_, err = svc.CreateUser(ctx, reqProdiInvalidScope, superadminID)
	if err == nil {
		t.Fatal("Scope ID prodi yang tidak valid (< 2 char) harus ditolak")
	}

	// Pastikan user tersimpan di DB
	savedProdi, err := userRepo.FindByEmail(ctx, "kaprodi.ti@univ.ac.id")
	if err != nil || savedProdi == nil {
		t.Fatal("User prodi harus tersimpan di DB")
	}
}

// 2. Guard Superadmin: Tidak boleh demote diri sendiri & tidak boleh hilangkan superadmin terakhir
func TestAdminService_SuperadminGuards(t *testing.T) {
	svc, _, _, _, superadminID := setupTestAdminService(t)
	ctx := context.Background()

	// A. Superadmin mencoba mencabut role superadmin dari dirinya sendiri -> ErrSelfDemoteSuperadmin
	reqDemote := dto.AdminUpdateRolesRequest{
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleKeuangan}, // Mau downgrade diri sendiri ke keuangan
		},
	}
	_, err := svc.UpdateUserRoles(ctx, superadminID, reqDemote, superadminID)
	if !errors.Is(err, ErrSelfDemoteSuperadmin) {
		t.Fatalf("Superadmin demote diri sendiri error = %v; expected ErrSelfDemoteSuperadmin", err)
	}

	// B. Superadmin mencoba menonaktifkan akun miliknya sendiri -> ErrSelfDeactivateSuperadmin
	reqDeactivate := dto.AdminUpdateStatusRequest{
		Status: model.UserStatusInactive,
	}
	_, err = svc.UpdateUserStatus(ctx, superadminID, reqDeactivate, superadminID)
	if !errors.Is(err, ErrSelfDeactivateSuperadmin) {
		t.Fatalf("Superadmin deactive diri sendiri error = %v; expected ErrSelfDeactivateSuperadmin", err)
	}
}

// 3. Test Internal API: GetUser, Batch, dan AddRole (mis. mahasiswa)
func TestAdminService_InternalAPI(t *testing.T) {
	svc, userRepo, _, _, superadminID := setupTestAdminService(t)
	ctx := context.Background()

	// Buat user calon mahasiswa
	userCalonID := uuid.New()
	userRepo.Create(ctx, &model.User{
		ID:       userCalonID,
		Email:    "calon@univ.ac.id",
		FullName: "Calon Maba",
		Status:   model.UserStatusActive,
	})

	// A. Internal GetUser
	u, err := svc.InternalGetUser(ctx, userCalonID)
	if err != nil {
		t.Fatalf("InternalGetUser error: %v", err)
	}
	if u.Email != "calon@univ.ac.id" {
		t.Errorf("InternalGetUser email = %s", u.Email)
	}

	// B. Internal Batch
	batch, err := svc.InternalGetUsersBatch(ctx, []uuid.UUID{userCalonID, superadminID})
	if err != nil {
		t.Fatalf("InternalGetUsersBatch error: %v", err)
	}
	if len(batch) != 2 {
		t.Errorf("InternalGetUsersBatch length = %d; expected 2", len(batch))
	}

	// C. Internal AddRole (mis. sistem pendaftaran menambahkan role mahasiswa saat lulus)
	addRoleReq := dto.AddUserRoleRequest{
		Role: model.RoleMahasiswa,
	}
	updated, err := svc.InternalAddRole(ctx, userCalonID, addRoleReq)
	if err != nil {
		t.Fatalf("InternalAddRole error: %v", err)
	}
	if len(updated.Roles) == 0 {
		t.Error("Roles pada user harus memuat role baru")
	}
}

// 4. Test Acceptance Criteria Milestone M2:
// Superadmin bisa membuat akun keuangan dan prodi (dengan scope),
// dan akun tersebut bisa login dengan role & scope yang benar di token.
func TestAdminService_M2_AcceptanceCriteria_CreatedUsersCanLoginWithRolesAndScopes(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		BcryptCost:    bcrypt.MinCost,
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		FrontendURL:   "http://localhost:5173",
	}

	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	resetRepo := newFakePasswordResetRepo()
	auditRepo := newFakeAuditRepo()
	sessionStore := newFakeSessionStore()
	rateLimiter := newFakeRateLimiter()
	otpRepo := newFakeOTPRepo()
	mailer := newFakeMailer()
	logger := zap.NewNop()

	tm, err := token.NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}

	adminSvc := NewAdminService(cfg, userRepo, roleRepo, resetRepo, auditRepo, sessionStore, mailer, logger)
	authSvc := NewAuthService(cfg, userRepo, roleRepo, otpRepo, resetRepo, auditRepo, tm, sessionStore, rateLimiter, mailer, logger)

	ctx := context.Background()
	superadminID := uuid.New()

	// 1. Superadmin buat user Keuangan
	resKeuangan, err := adminSvc.CreateUser(ctx, dto.AdminCreateUserRequest{
		Email:    "keuangan@kampus.ac.id",
		FullName: "Staf Keuangan Kampus",
		Phone:    "081234567890",
		Password: "PasswordKeuangan1!",
		Roles: []dto.RoleAssignmentDTO{
			{Code: model.RoleKeuangan},
		},
	}, superadminID)
	if err != nil {
		t.Fatalf("Superadmin gagal buat user keuangan: %v", err)
	}
	if resKeuangan.Status != "active" {
		t.Fatalf("User keuangan harus langsung active, didapat: %s", resKeuangan.Status)
	}

	// 2. User Keuangan Login -> Cek token claims
	loginKeuangan, err := authSvc.Login(ctx, dto.LoginRequest{
		Email:    "keuangan@kampus.ac.id",
		Password: "PasswordKeuangan1!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login keuangan gagal: %v", err)
	}

	claimsKeuangan, err := tm.ValidateAccessToken(loginKeuangan.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken keuangan gagal: %v", err)
	}
	if len(claimsKeuangan.Roles) != 1 || claimsKeuangan.Roles[0] != model.RoleKeuangan {
		t.Fatalf("Claim role token keuangan salah: %v; expected ['keuangan']", claimsKeuangan.Roles)
	}

	// 3. Superadmin buat user Prodi dengan Scope
	resProdi, err := adminSvc.CreateUser(ctx, dto.AdminCreateUserRequest{
		Email:    "kaprodi.ti@kampus.ac.id",
		FullName: "Kaprodi Teknik Informatika",
		Phone:    "081234567899",
		Password: "PasswordProdi1!",
		Roles: []dto.RoleAssignmentDTO{
			{
				Code:      model.RoleProdi,
				ScopeType: "prodi",
				ScopeID:   "TI",
			},
		},
	}, superadminID)
	if err != nil {
		t.Fatalf("Superadmin gagal buat user prodi: %v", err)
	}
	if resProdi.Status != "active" {
		t.Fatalf("User prodi harus langsung active, didapat: %s", resProdi.Status)
	}

	// 4. User Prodi Login -> Cek token claims
	loginProdi, err := authSvc.Login(ctx, dto.LoginRequest{
		Email:    "kaprodi.ti@kampus.ac.id",
		Password: "PasswordProdi1!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login prodi gagal: %v", err)
	}

	claimsProdi, err := tm.ValidateAccessToken(loginProdi.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken prodi gagal: %v", err)
	}
	if len(claimsProdi.Roles) != 1 || claimsProdi.Roles[0] != model.RoleProdi {
		t.Fatalf("Claim role token prodi salah: %v; expected ['prodi']", claimsProdi.Roles)
	}
	if len(claimsProdi.Scopes) != 1 || claimsProdi.Scopes[0].ID != "TI" {
		t.Fatalf("Claim scopes token prodi salah: %v; expected scope ID 'TI'", claimsProdi.Scopes)
	}
}

