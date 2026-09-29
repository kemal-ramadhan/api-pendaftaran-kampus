package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/service"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// In-Memory Fakes untuk Controller Test
type mockUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*model.User
}

func (r *mockUserRepo) Create(ctx context.Context, u *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	cpy := *u
	r.users[u.ID] = &cpy
	return nil
}

func (r *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[id]; ok {
		cpy := *u
		return &cpy, nil
	}
	return nil, nil
}

func (r *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	for _, u := range r.users {
		if strings.ToLower(strings.TrimSpace(u.Email)) == clean {
			cpy := *u
			return &cpy, nil
		}
	}
	return nil, nil
}

func (r *mockUserRepo) Update(ctx context.Context, u *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cpy := *u
	r.users[u.ID] = &cpy
	return nil
}

func (r *mockUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[userID]; ok {
		u.PasswordHash = passwordHash
	}
	return nil
}

func (r *mockUserRepo) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (r *mockUserRepo) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	return nil
}

func (r *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	u, err := r.FindByEmail(ctx, email)
	return u != nil, err
}

func (r *mockUserRepo) FindUsers(ctx context.Context, role, status, q string, page, perPage int) ([]model.User, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []model.User
	for _, u := range r.users {
		cpy := *u
		res = append(res, cpy)
	}
	return res, int64(len(res)), nil
}

func (r *mockUserRepo) FindBatch(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []model.User
	for _, id := range ids {
		if u, ok := r.users[id]; ok {
			cpy := *u
			res = append(res, cpy)
		}
	}
	return res, nil
}

type mockRoleRepo struct {
	roles map[string]*model.Role
}

func (r *mockRoleRepo) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	if role, ok := r.roles[code]; ok {
		return role, nil
	}
	return nil, nil
}

func (r *mockRoleRepo) FindAll(ctx context.Context) ([]model.Role, error) {
	return nil, nil
}

func (r *mockRoleRepo) AssignRole(ctx context.Context, ur *model.UserRole) error {
	return nil
}

func (r *mockRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	return nil, nil
}

func (r *mockRoleRepo) RemoveUserRoles(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (r *mockRoleRepo) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roles []model.UserRole) error {
	return nil
}

func (r *mockRoleRepo) CountActiveSuperadmins(ctx context.Context) (int64, error) {
	return 1, nil
}

func (r *mockRoleRepo) RemoveUserRoleByCode(ctx context.Context, userID uuid.UUID, code string) error {
	return nil
}

func (r *mockRoleRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return []string{"dashboard.overview.read"}, nil
}

func (r *mockRoleRepo) GetAllPermissions(ctx context.Context) ([]model.Permission, error) {
	return []model.Permission{
		{Code: "dashboard.overview.read", Module: "dashboard", Description: "Baca overview"},
	}, nil
}

func (r *mockRoleRepo) GetRolePermissions(ctx context.Context, roleCode string) ([]string, error) {
	return []string{"dashboard.overview.read"}, nil
}

func (r *mockRoleRepo) UpdateRolePermissions(ctx context.Context, roleCode string, permissionCodes []string) error {
	return nil
}



type mockOTPRepo struct {
	mu   sync.Mutex
	otps map[uuid.UUID]*model.UserOTP
}

func (r *mockOTPRepo) Create(ctx context.Context, o *model.UserOTP) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	cpy := *o
	r.otps[o.ID] = &cpy
	return nil
}

func (r *mockOTPRepo) FindLatestActive(ctx context.Context, userID uuid.UUID, purpose string) (*model.UserOTP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var latest *model.UserOTP
	for _, o := range r.otps {
		if o.UserID == userID && o.Purpose == purpose && o.ConsumedAt == nil {
			if latest == nil || o.CreatedAt.After(latest.CreatedAt) {
				latest = o
			}
		}
	}
	if latest == nil {
		return nil, nil
	}
	cpy := *latest
	return &cpy, nil
}

func (r *mockOTPRepo) IncrementAttempts(ctx context.Context, otpID uuid.UUID) (int16, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.otps[otpID]; ok {
		o.Attempts++
		return o.Attempts, nil
	}
	return 0, errors.New("not found")
}

func (r *mockOTPRepo) MarkConsumed(ctx context.Context, otpID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.otps[otpID]; ok {
		now := time.Now()
		o.ConsumedAt = &now
	}
	return nil
}

func (r *mockOTPRepo) InvalidateActiveOTPs(ctx context.Context, userID uuid.UUID, purpose string) error {
	return nil
}

type mockResetRepo struct{}

func (r *mockResetRepo) Create(ctx context.Context, pr *model.PasswordReset) error                   { return nil }
func (r *mockResetRepo) FindByTokenHash(ctx context.Context, hash string) (*model.PasswordReset, error) { return nil, nil }
func (r *mockResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error                           { return nil }
func (r *mockResetRepo) InvalidateActiveTokens(ctx context.Context, id uuid.UUID) error             { return nil }

type mockAuditRepo struct{}

func (r *mockAuditRepo) Create(ctx context.Context, l *model.AuditLog) error { return nil }

type mockRateLimiter struct {
	mu          sync.Mutex
	cooldowns   map[string]bool
	lockedEmail string
}

func (l *mockRateLimiter) IsLoginLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return email == l.lockedEmail, 15 * time.Minute, nil
}
func (l *mockRateLimiter) RecordLoginFailure(ctx context.Context, email string) (bool, int, error) {
	return false, 4, nil
}
func (l *mockRateLimiter) ResetLoginFailure(ctx context.Context, email string) error { return nil }
func (l *mockRateLimiter) CheckResendOTPCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cooldowns[email], 60 * time.Second, nil
}
func (l *mockRateLimiter) SetResendOTPCooldown(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cooldowns[email] = true
	return nil
}
func (l *mockRateLimiter) CheckForgotPasswordCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	return false, 0, nil
}
func (l *mockRateLimiter) SetForgotPasswordCooldown(ctx context.Context, email string) error {
	return nil
}

type mockMailer struct {
	lastOTP string
}

func (m *mockMailer) SendOTP(ctx context.Context, toEmail, toName, otpCode string) error {
	m.lastOTP = otpCode
	return nil
}
func (m *mockMailer) SendPasswordReset(ctx context.Context, toEmail, toName, resetLink string) error {
	return nil
}

type mockSessionStore struct {
	mu       sync.Mutex
	sessions map[string]string
}

func (s *mockSessionStore) SaveSession(ctx context.Context, userID uuid.UUID, sessionID, jti string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[userID.String()+":"+sessionID] = jti
	return nil
}
func (s *mockSessionStore) ValidateSession(ctx context.Context, userID uuid.UUID, sessionID, jti string) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.sessions[userID.String()+":"+sessionID]
	if !ok {
		return false, false, nil
	}
	if val != jti {
		return false, true, nil
	}
	return true, false, nil
}
func (s *mockSessionStore) RotateSession(ctx context.Context, userID uuid.UUID, sessionID, newJTI string, ttl time.Duration) error {
	return nil
}
func (s *mockSessionStore) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	return nil
}
func (s *mockSessionStore) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (s *mockSessionStore) AddToDenylist(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}
func (s *mockSessionStore) IsDenylisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

func setupTestRouter(t *testing.T) (*gin.Engine, *mockUserRepo, *mockOTPRepo, *mockRateLimiter, *mockMailer, token.TokenManager) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		AppEnv:        "test",
		BcryptCost:    bcrypt.MinCost,
		OTPTTL:        10 * time.Minute,
		OTPSecret:     "secret-32-bytes-test-key-here12",
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	userRepo := &mockUserRepo{users: make(map[uuid.UUID]*model.User)}
	roleRepo := &mockRoleRepo{roles: map[string]*model.Role{
		model.RoleCalonMahasiswa: {ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"},
	}}
	otpRepo := &mockOTPRepo{otps: make(map[uuid.UUID]*model.UserOTP)}
	resetRepo := &mockResetRepo{}
	auditRepo := &mockAuditRepo{}
	rateLimiter := &mockRateLimiter{cooldowns: make(map[string]bool)}
	mailer := &mockMailer{}
	sessionStore := &mockSessionStore{sessions: make(map[string]string)}

	logger := zap.NewNop()
	tm, err := token.NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewTokenManager error: %v", err)
	}

	authSvc := service.NewAuthService(
		cfg,
		userRepo,
		roleRepo,
		otpRepo,
		resetRepo,
		auditRepo,
		tm,
		sessionStore,
		rateLimiter,
		mailer,
		logger,
	)

	authCtrl := NewAuthController(authSvc, tm)
	healthCtrl := NewHealthController(nil, nil, "test")

	r := gin.New()
	r.GET("/health", healthCtrl.LivenessCheck)
	r.GET("/health/ready", healthCtrl.HealthCheck)
	r.GET("/auth/health", healthCtrl.HealthCheck)
	r.GET("/.well-known/jwks.json", authCtrl.GetJWKS)

	auth := r.Group("/auth")
	{
		auth.POST("/register", authCtrl.Register)
		auth.POST("/verify", authCtrl.VerifyOTP)
		auth.POST("/resend-otp", authCtrl.ResendOTP)
		auth.POST("/login", authCtrl.Login)
		auth.POST("/refresh", authCtrl.Refresh)
		auth.POST("/forgot-password", authCtrl.ForgotPassword)
		auth.POST("/reset-password", authCtrl.ResetPassword)
	}

	authProtected := auth.Group("")
	authProtected.Use(middleware.AuthMiddleware(tm, sessionStore))
	{
		authProtected.GET("/me", authCtrl.GetMe)
		authProtected.PUT("/profile", authCtrl.UpdateProfile)
		authProtected.PUT("/password", authCtrl.ChangePassword)
		authProtected.POST("/logout", authCtrl.Logout)
		authProtected.POST("/logout-all", authCtrl.LogoutAll)
	}

	return r, userRepo, otpRepo, rateLimiter, mailer, tm
}

// 1. Test POST /auth/register
func TestController_Register(t *testing.T) {
	r, _, _, _, _, _ := setupTestRouter(t)

	// Valid payload
	body := `{"full_name":"Ahmad Fauzi","email":"ahmad@example.com","phone":"081234567890","password":"Password123!","password_confirmation":"Password123!"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /auth/register status = %d; expected %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var res response.SuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Unmarshal response error: %v", err)
	}
	if !res.Success {
		t.Errorf("Response success = false; expected true")
	}

	// Invalid password mismatch -> 422
	badBody := `{"full_name":"Ahmad","email":"ahmad2@example.com","phone":"0812","password":"Password123!","password_confirmation":"BedaPassword!"}`
	req2, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(badBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnprocessableEntity {
		t.Errorf("POST /auth/register mismatch status = %d; expected %d", w2.Code, http.StatusUnprocessableEntity)
	}
}

// 2. Test POST /auth/verify (Mendukung baik string maupun number OTP)
func TestController_VerifyOTP_Flexibility(t *testing.T) {
	r, userRepo, otpRepo, _, _, _ := setupTestRouter(t)

	userID := uuid.New()
	roleCalon := model.Role{ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"}
	emptyScope := ""
	userRepo.Create(context.Background(), &model.User{
		ID:       userID,
		Email:    "verifyflex@example.com",
		FullName: "Flexi User",
		Status:   model.UserStatusPending,
		UserRoles: []model.UserRole{
			{UserID: userID, RoleID: 1, Role: roleCalon, ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	})

	code := "123456"
	codeHash := utils.HashOTP(code, "secret-32-bytes-test-key-here12")
	otpRepo.Create(context.Background(), &model.UserOTP{
		UserID:      userID,
		Purpose:     model.OTPPurposeEmailVerification,
		CodeHash:    codeHash,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Attempts:    0,
		MaxAttempts: 5,
	})

	// Test A: Kirim OTP sebagai number 123456 (sesuai contoh di DECISIONS.md)
	numberBody := `{"email":"verifyflex@example.com","otp":123456}`
	reqA, _ := http.NewRequest(http.MethodPost, "/auth/verify", bytes.NewBufferString(numberBody))
	reqA.Header.Set("Content-Type", "application/json")
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)

	if wA.Code != http.StatusOK {
		t.Fatalf("POST /auth/verify dengan OTP number gagal, status = %d, body: %s", wA.Code, wA.Body.String())
	}

	var resA response.SuccessResponse
	json.Unmarshal(wA.Body.Bytes(), &resA)
	if !resA.Success {
		t.Errorf("Response success = false")
	}
}

// 3. Test POST /auth/resend-otp
func TestController_ResendOTP(t *testing.T) {
	r, userRepo, _, rateLimiter, _, _ := setupTestRouter(t)

	userRepo.Create(context.Background(), &model.User{
		ID:       uuid.New(),
		Email:    "resend@example.com",
		FullName: "Resend User",
		Status:   model.UserStatusPending,
	})

	// Panggilan pertama -> 200 OK
	body := `{"email":"resend@example.com"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/resend-otp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /auth/resend-otp status = %d; expected %d", w.Code, http.StatusOK)
	}

	// Panggilan kedua saat cooldown -> 429
	rateLimiter.cooldowns["resend@example.com"] = true
	req2, _ := http.NewRequest(http.MethodPost, "/auth/resend-otp", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("POST /auth/resend-otp cooldown status = %d; expected %d", w2.Code, http.StatusTooManyRequests)
	}
}

// 4. Test POST /auth/login (termasuk verifikasi response regresi celah 8)
func TestController_Login(t *testing.T) {
	r, userRepo, _, rateLimiter, _, _ := setupTestRouter(t)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)
	roleCalon := model.Role{ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"}
	emptyScope := ""
	userID := uuid.New()
	userRepo.Create(context.Background(), &model.User{
		ID:           userID,
		Email:        "login@example.com",
		PasswordHash: string(hashed),
		FullName:     "Login User",
		Status:       model.UserStatusActive,
		UserRoles: []model.UserRole{
			{UserID: userID, RoleID: 1, Role: roleCalon, ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	})

	// Login sukses -> 200 OK
	body := `{"email":"login@example.com","password":"Password123!"}`
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /auth/login status = %d; expected 200, body: %s", w.Code, w.Body.String())
	}

	// REGRESI CELAH NO. 8 (Response level):
	// Uji bahwa response saat email tidak ada vs password salah adalah IDENTIK
	reqNotFound, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"gaada@example.com","password":"Password123!"}`))
	reqNotFound.Header.Set("Content-Type", "application/json")
	wNF := httptest.NewRecorder()
	r.ServeHTTP(wNF, reqNotFound)

	reqWrongPass, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"login@example.com","password":"SalahPassword!"}`))
	reqWrongPass.Header.Set("Content-Type", "application/json")
	wWP := httptest.NewRecorder()
	r.ServeHTTP(wWP, reqWrongPass)

	if wNF.Code != http.StatusUnauthorized || wWP.Code != http.StatusUnauthorized {
		t.Errorf("Kedua kasus gagal harus berstatus 401 Unauthorized (%d vs %d)", wNF.Code, wWP.Code)
	}

	var resNF, resWP response.ErrorResponse
	json.Unmarshal(wNF.Body.Bytes(), &resNF)
	json.Unmarshal(wWP.Body.Bytes(), &resWP)

	if resNF.Message != resWP.Message {
		t.Errorf("Celah No. 8 Terbuka di level HTTP: pesan error berbeda! [%s] vs [%s]", resNF.Message, resWP.Message)
	}
	if resNF.ErrorCode != resWP.ErrorCode {
		t.Errorf("ErrorCode berbeda: [%s] vs [%s]", resNF.ErrorCode, resWP.ErrorCode)
	}

	// Login saat terkunci (Lockout 5x gagal) -> 429
	rateLimiter.lockedEmail = "terkunci@example.com"
	reqLocked, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"terkunci@example.com","password":"Password123!"}`))
	reqLocked.Header.Set("Content-Type", "application/json")
	wLocked := httptest.NewRecorder()
	r.ServeHTTP(wLocked, reqLocked)

	if wLocked.Code != http.StatusTooManyRequests {
		t.Errorf("POST /auth/login lockout status = %d; expected 429", wLocked.Code)
	}
}

// 5. Test GET /auth/me & Middleware Auth
func TestController_GetMe(t *testing.T) {
	r, userRepo, _, _, _, tm := setupTestRouter(t)

	userID := uuid.New()
	roleCalon := model.Role{ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"}
	emptyScope := ""
	userRepo.Create(context.Background(), &model.User{
		ID:       userID,
		Email:    "me_endpoint@example.com",
		FullName: "Endpoint User Me",
		Status:   model.UserStatusActive,
		UserRoles: []model.UserRole{
			{UserID: userID, RoleID: 1, Role: roleCalon, ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	})

	// 1. Request tanpa token -> 401
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("GET /auth/me tanpa token harus 401; didapat %d", wUnauth.Code)
	}

	// 2. Request dengan valid access token -> 200
	tokenStr, _, _, err := tm.GenerateAccessToken(userID, "me_endpoint@example.com", "Endpoint User Me", []string{"calon_mahasiswa"}, nil, nil, uuid.NewString())
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}

	reqAuth, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+tokenStr)
	wAuth := httptest.NewRecorder()
	r.ServeHTTP(wAuth, reqAuth)

	if wAuth.Code != http.StatusOK {
		t.Fatalf("GET /auth/me dengan token valid gagal, status = %d, body: %s", wAuth.Code, wAuth.Body.String())
	}

	var res response.SuccessResponse
	json.Unmarshal(wAuth.Body.Bytes(), &res)
	if !res.Success {
		t.Errorf("GET /auth/me success = false")
	}

	// 3. REGRESI CELAH NO. 7 (Middleware level):
	// Request dengan refresh token ke endpoint terproteksi -> HARUS DITOLAK (401)!
	refreshStr, _, _, _ := tm.GenerateRefreshToken(userID, uuid.NewString())
	reqRefresh, _ := http.NewRequest(http.MethodGet, "/auth/me", nil)
	reqRefresh.Header.Set("Authorization", "Bearer "+refreshStr)
	wRefresh := httptest.NewRecorder()
	r.ServeHTTP(wRefresh, reqRefresh)

	if wRefresh.Code != http.StatusUnauthorized {
		t.Fatalf("Celah No. 7 Terbuka: Refresh token berhasil mengakses endpoint GET /auth/me! Status = %d", wRefresh.Code)
	}
}

// 6. Test Health Endpoints
func TestController_Health(t *testing.T) {
	r, _, _, _, _, _ := setupTestRouter(t)

	// Liveness probe harus selalu 200 OK
	reqLive, _ := http.NewRequest(http.MethodGet, "/health", nil)
	wLive := httptest.NewRecorder()
	r.ServeHTTP(wLive, reqLive)
	if wLive.Code != http.StatusOK {
		t.Errorf("GET /health status = %d; expected 200", wLive.Code)
	}

	// Readiness probe tanpa DB/Redis offline harus 503 Service Unavailable
	readyPaths := []string{"/health/ready", "/auth/health"}
	for _, p := range readyPaths {
		req, _ := http.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s status = %d; expected 503 (saat DB/Redis offline)", p, w.Code)
		}
	}
}

// 7. Test GET /.well-known/jwks.json
func TestController_JWKS(t *testing.T) {
	r, _, _, _, _, _ := setupTestRouter(t)

	req, _ := http.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /.well-known/jwks.json status = %d; expected 200", w.Code)
	}

	var jwks token.JWKSResponse
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("Unmarshal JWKS error: %v", err)
	}

	if len(jwks.Keys) != 1 {
		t.Errorf("JWKS keys length = %d; expected 1", len(jwks.Keys))
	}
	if jwks.Keys[0].Alg != "RS256" {
		t.Errorf("JWKS key alg = %s; expected RS256", jwks.Keys[0].Alg)
	}
}

// 8. Test Regresi Celah Keamanan No. 4, 5, dan 8 (Forgot Password)
func TestController_SecurityRegressions_Celah4_5_8(t *testing.T) {
	r, userRepo, _, _, _, tm := setupTestRouter(t)

	// Persiapkan 2 user untuk uji IDOR (Celah 5)
	userA := &model.User{
		ID:       uuid.New(),
		Email:    "usera@example.com",
		FullName: "User Asli",
		Phone:    "+628111111111",
		Status:   model.UserStatusActive,
	}
	userB := &model.User{
		ID:       uuid.New(),
		Email:    "userb@example.com",
		FullName: "Target IDOR",
		Phone:    "+628222222222",
		Status:   model.UserStatusActive,
	}
	_ = userRepo.Create(context.Background(), userA)
	_ = userRepo.Create(context.Background(), userB)

	tokenA, _, _, _ := tm.GenerateAccessToken(userA.ID, userA.Email, userA.FullName, []string{"calon_mahasiswa"}, nil, nil, uuid.NewString())

	// CELAH NO. 4: PUT /auth/change-role dihapus; user tidak boleh bisa ubah role sendiri
	reqChangeRole, _ := http.NewRequest(http.MethodPut, "/auth/change-role", strings.NewReader(`{"role":"superadmin"}`))
	reqChangeRole.Header.Set("Authorization", "Bearer "+tokenA)
	reqChangeRole.Header.Set("Content-Type", "application/json")
	wChangeRole := httptest.NewRecorder()
	r.ServeHTTP(wChangeRole, reqChangeRole)
	if wChangeRole.Code != http.StatusNotFound {
		t.Fatalf("Celah No. 4 Terbuka: PUT /auth/change-role harus 404 (dihapus), didapat: %d", wChangeRole.Code)
	}

	// CELAH NO. 5: PUT /auth/profile/:id dihapus (mencegah IDOR)
	idorURL := "/auth/profile/" + userB.ID.String()
	reqIDOR, _ := http.NewRequest(http.MethodPut, idorURL, strings.NewReader(`{"full_name":"Hacked Name"}`))
	reqIDOR.Header.Set("Authorization", "Bearer "+tokenA)
	reqIDOR.Header.Set("Content-Type", "application/json")
	wIDOR := httptest.NewRecorder()
	r.ServeHTTP(wIDOR, reqIDOR)
	if wIDOR.Code != http.StatusNotFound {
		t.Fatalf("Celah No. 5 Terbuka: PUT /auth/profile/:id harus 404 (mencegah IDOR), didapat: %d", wIDOR.Code)
	}

	// Pastikan profil userB tidak berubah sama sekali
	checkB, _ := userRepo.FindByID(context.Background(), userB.ID)
	if checkB.FullName != "Target IDOR" {
		t.Fatalf("Celah No. 5 Terbuka: Profil user B termodifikasi oleh user A!")
	}

	// CELAH NO. 8 (Forgot Password): Respons identik apakah email terdaftar atau tidak
	// Kasus A: Email ada
	reqFPA, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(`{"email":"usera@example.com"}`))
	reqFPA.Header.Set("Content-Type", "application/json")
	wFPA := httptest.NewRecorder()
	r.ServeHTTP(wFPA, reqFPA)

	// Kasus B: Email tidak ada
	reqFPB, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(`{"email":"tidakada@example.com"}`))
	reqFPB.Header.Set("Content-Type", "application/json")
	wFPB := httptest.NewRecorder()
	r.ServeHTTP(wFPB, reqFPB)

	if wFPA.Code != http.StatusOK || wFPB.Code != http.StatusOK {
		t.Fatalf("Forgot password harus selalu mengembalikan 200 OK, didapat A: %d, B: %d", wFPA.Code, wFPB.Code)
	}

	var resA, resB response.SuccessResponse
	_ = json.Unmarshal(wFPA.Body.Bytes(), &resA)
	_ = json.Unmarshal(wFPB.Body.Bytes(), &resB)
	if resA.Message != resB.Message {
		t.Fatalf("Celah No. 8 Terbuka: Pesan respon forgot password membocorkan keberadaan email! [%s] vs [%s]", resA.Message, resB.Message)
	}
}

