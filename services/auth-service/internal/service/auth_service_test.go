package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// --- Fake In-Memory Repositories untuk Hermetic Unit Testing ---

type fakeUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[uuid.UUID]*model.User)}
}

func (r *fakeUserRepo) Create(ctx context.Context, u *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	cpy := *u
	r.users[u.ID] = &cpy
	return nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	cpy := *u
	return &cpy, nil
}

func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
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

func (r *fakeUserRepo) Update(ctx context.Context, u *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cpy := *u
	r.users[u.ID] = &cpy
	return nil
}

func (r *fakeUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[userID]; ok {
		u.PasswordHash = passwordHash
		now := time.Now()
		u.PasswordChangedAt = &now
	}
	return nil
}

func (r *fakeUserRepo) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[userID]; ok {
		now := time.Now()
		u.LastLoginAt = &now
	}
	return nil
}

func (r *fakeUserRepo) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[userID]; ok {
		u.Status = status
	}
	return nil
}

func (r *fakeUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	u, err := r.FindByEmail(ctx, email)
	return u != nil, err
}

func (r *fakeUserRepo) FindUsers(ctx context.Context, role, status, q string, page, perPage int) ([]model.User, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []model.User
	for _, u := range r.users {
		if status != "" && u.Status != status {
			continue
		}
		if q != "" {
			search := strings.ToLower(q)
			if !strings.Contains(strings.ToLower(u.FullName), search) && !strings.Contains(strings.ToLower(u.Email), search) {
				continue
			}
		}
		cpy := *u
		res = append(res, cpy)
	}
	return res, int64(len(res)), nil
}

func (r *fakeUserRepo) FindBatch(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []model.User
	idMap := make(map[uuid.UUID]bool)
	for _, id := range ids {
		idMap[id] = true
	}
	for _, u := range r.users {
		if idMap[u.ID] {
			cpy := *u
			res = append(res, cpy)
		}
	}
	return res, nil
}

type fakeRoleRepo struct {
	mu        sync.Mutex
	roles     map[string]*model.Role
	userRoles map[uuid.UUID][]model.UserRole
}

func newFakeRoleRepo() *fakeRoleRepo {
	return &fakeRoleRepo{
		roles: map[string]*model.Role{
			model.RoleCalonMahasiswa: {ID: 1, Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa"},
			model.RoleMahasiswa:      {ID: 2, Code: model.RoleMahasiswa, Name: "Mahasiswa"},
			model.RoleAdminPMB:       {ID: 3, Code: model.RoleAdminPMB, Name: "Admin PMB"},
			model.RoleKeuangan:       {ID: 4, Code: model.RoleKeuangan, Name: "Keuangan"},
			model.RoleDosen:          {ID: 5, Code: model.RoleDosen, Name: "Dosen"},
			model.RoleProdi:          {ID: 6, Code: model.RoleProdi, Name: "Prodi"},
			model.RolePimpinan:       {ID: 7, Code: model.RolePimpinan, Name: "Pimpinan"},
			model.RoleSuperadmin:     {ID: 8, Code: model.RoleSuperadmin, Name: "Superadmin"},
		},
		userRoles: make(map[uuid.UUID][]model.UserRole),
	}
}

func (r *fakeRoleRepo) FindByCode(ctx context.Context, code string) (*model.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role, ok := r.roles[code]; ok {
		cpy := *role
		return &cpy, nil
	}
	return nil, nil
}

func (r *fakeRoleRepo) FindAll(ctx context.Context) ([]model.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]model.Role, 0, len(r.roles))
	for _, role := range r.roles {
		res = append(res, *role)
	}
	return res, nil
}

func (r *fakeRoleRepo) AssignRole(ctx context.Context, ur *model.UserRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userRoles[ur.UserID] = append(r.userRoles[ur.UserID], *ur)
	return nil
}

func (r *fakeRoleRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]model.UserRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.userRoles[userID], nil
}

func (r *fakeRoleRepo) RemoveUserRoles(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.userRoles, userID)
	return nil
}

func (r *fakeRoleRepo) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roles []model.UserRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userRoles[userID] = roles
	return nil
}

func (r *fakeRoleRepo) CountActiveSuperadmins(ctx context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for _, list := range r.userRoles {
		for _, ur := range list {
			if ur.Role.Code == model.RoleSuperadmin {
				count++
				break
			}
		}
	}
	return count, nil
}

func (r *fakeRoleRepo) RemoveUserRoleByCode(ctx context.Context, userID uuid.UUID, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.userRoles[userID]
	filtered := make([]model.UserRole, 0, len(current))
	for _, ur := range current {
		if ur.Role.Code != code {
			filtered = append(filtered, ur)
		}
	}
	r.userRoles[userID] = filtered
	return nil
}



type fakeOTPRepo struct {
	mu   sync.Mutex
	otps map[uuid.UUID]*model.UserOTP
}

func newFakeOTPRepo() *fakeOTPRepo {
	return &fakeOTPRepo{otps: make(map[uuid.UUID]*model.UserOTP)}
}

func (r *fakeOTPRepo) Create(ctx context.Context, o *model.UserOTP) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	cpy := *o
	r.otps[o.ID] = &cpy
	return nil
}

func (r *fakeOTPRepo) FindLatestActive(ctx context.Context, userID uuid.UUID, purpose string) (*model.UserOTP, error) {
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

func (r *fakeOTPRepo) IncrementAttempts(ctx context.Context, otpID uuid.UUID) (int16, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.otps[otpID]; ok {
		o.Attempts++
		return o.Attempts, nil
	}
	return 0, errors.New("otp not found")
}

func (r *fakeOTPRepo) MarkConsumed(ctx context.Context, otpID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.otps[otpID]; ok {
		now := time.Now()
		o.ConsumedAt = &now
	}
	return nil
}

func (r *fakeOTPRepo) InvalidateActiveOTPs(ctx context.Context, userID uuid.UUID, purpose string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, o := range r.otps {
		if o.UserID == userID && o.Purpose == purpose && o.ConsumedAt == nil {
			o.ConsumedAt = &now
		}
	}
	return nil
}

type fakePasswordResetRepo struct {
	mu     sync.Mutex
	resets map[uuid.UUID]*model.PasswordReset
}

func newFakePasswordResetRepo() *fakePasswordResetRepo {
	return &fakePasswordResetRepo{resets: make(map[uuid.UUID]*model.PasswordReset)}
}

func (r *fakePasswordResetRepo) Create(ctx context.Context, pr *model.PasswordReset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	cpy := *pr
	r.resets[pr.ID] = &cpy
	return nil
}

func (r *fakePasswordResetRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pr := range r.resets {
		if pr.TokenHash == tokenHash {
			cpy := *pr
			return &cpy, nil
		}
	}
	return nil, nil
}

func (r *fakePasswordResetRepo) MarkUsed(ctx context.Context, resetID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pr, ok := r.resets[resetID]; ok {
		now := time.Now()
		pr.UsedAt = &now
	}
	return nil
}

func (r *fakePasswordResetRepo) InvalidateActiveTokens(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, pr := range r.resets {
		if pr.UserID == userID && pr.UsedAt == nil {
			pr.UsedAt = &now
		}
	}
	return nil
}

type fakeAuditRepo struct {
	mu   sync.Mutex
	logs []*model.AuditLog
}

func newFakeAuditRepo() *fakeAuditRepo {
	return &fakeAuditRepo{logs: make([]*model.AuditLog, 0)}
}

func (r *fakeAuditRepo) Create(ctx context.Context, l *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, l)
	return nil
}

type fakeRateLimiter struct {
	mu                 sync.Mutex
	lockedEmails       map[string]bool
	resendCooldowns    map[string]bool
	forgotCooldowns    map[string]bool
	failCounts         map[string]int
}

func newFakeRateLimiter() *fakeRateLimiter {
	return &fakeRateLimiter{
		lockedEmails:    make(map[string]bool),
		resendCooldowns: make(map[string]bool),
		forgotCooldowns: make(map[string]bool),
		failCounts:      make(map[string]int),
	}
}

func (l *fakeRateLimiter) IsLoginLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	if l.lockedEmails[clean] {
		return true, 15 * time.Minute, nil
	}
	return false, 0, nil
}

func (l *fakeRateLimiter) RecordLoginFailure(ctx context.Context, email string) (bool, int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	l.failCounts[clean]++
	if l.failCounts[clean] >= 5 {
		l.lockedEmails[clean] = true
		return true, 0, nil
	}
	return false, 5 - l.failCounts[clean], nil
}

func (l *fakeRateLimiter) ResetLoginFailure(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	delete(l.failCounts, clean)
	delete(l.lockedEmails, clean)
	return nil
}

func (l *fakeRateLimiter) CheckResendOTPCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	return l.resendCooldowns[clean], 60 * time.Second, nil
}

func (l *fakeRateLimiter) SetResendOTPCooldown(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	l.resendCooldowns[clean] = true
	return nil
}

func (l *fakeRateLimiter) CheckForgotPasswordCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	return l.forgotCooldowns[clean], 60 * time.Second, nil
}

func (l *fakeRateLimiter) SetForgotPasswordCooldown(ctx context.Context, email string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	clean := strings.ToLower(strings.TrimSpace(email))
	l.forgotCooldowns[clean] = true
	return nil
}

type fakeMailer struct {
	mu           sync.Mutex
	sentOTPs     map[string]string // email -> otpCode
	sentResets   map[string]string // email -> resetLink
}

func newFakeMailer() *fakeMailer {
	return &fakeMailer{
		sentOTPs:   make(map[string]string),
		sentResets: make(map[string]string),
	}
}

func (m *fakeMailer) SendOTP(ctx context.Context, toEmail, toName, otpCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentOTPs[toEmail] = otpCode
	return nil
}

func (m *fakeMailer) SendPasswordReset(ctx context.Context, toEmail, toName, resetLink string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentResets[toEmail] = resetLink
	return nil
}

func (m *fakeMailer) GetLastOTP(email string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentOTPs[email]
}

type fakeSessionStore struct {
	mu          sync.Mutex
	sessions    map[string]string // "userID:sid" -> jti
	denylist    map[string]bool
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions: make(map[string]string),
		denylist: make(map[string]bool),
	}
}

func (s *fakeSessionStore) SaveSession(ctx context.Context, userID uuid.UUID, sessionID, jti string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID.String() + ":" + sessionID
	s.sessions[key] = jti
	return nil
}

func (s *fakeSessionStore) ValidateSession(ctx context.Context, userID uuid.UUID, sessionID, jti string) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID.String() + ":" + sessionID
	val, ok := s.sessions[key]
	if !ok {
		return false, false, nil
	}
	if val != jti {
		return false, true, nil // Reuse detected
	}
	return true, false, nil
}

func (s *fakeSessionStore) RotateSession(ctx context.Context, userID uuid.UUID, sessionID, newJTI string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID.String() + ":" + sessionID
	s.sessions[key] = newJTI
	return nil
}

func (s *fakeSessionStore) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := userID.String() + ":" + sessionID
	delete(s.sessions, key)
	return nil
}

func (s *fakeSessionStore) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := userID.String() + ":"
	for k := range s.sessions {
		if strings.HasPrefix(k, prefix) {
			delete(s.sessions, k)
		}
	}
	return nil
}

func (s *fakeSessionStore) AddToDenylist(ctx context.Context, jti string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.denylist[jti] = true
	return nil
}

func (s *fakeSessionStore) IsDenylisted(ctx context.Context, jti string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.denylist[jti], nil
}

// Helper untuk inisialisasi AuthService dengan fake dependencies
func setupTestAuthService(t *testing.T) (
	AuthService,
	*fakeUserRepo,
	*fakeRoleRepo,
	*fakeOTPRepo,
	*fakeRateLimiter,
	*fakeMailer,
	token.TokenManager,
	*fakeSessionStore,
) {
	cfg := &config.Config{
		AppEnv:        "test",
		BcryptCost:    bcrypt.MinCost, // Cepat untuk test
		OTPTTL:        10 * time.Minute,
		OTPSecret:     "unit-test-otp-secret-key-32bytes",
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	otpRepo := newFakeOTPRepo()
	resetRepo := newFakePasswordResetRepo()
	auditRepo := newFakeAuditRepo()
	rateLimiter := newFakeRateLimiter()
	mailer := newFakeMailer()
	sessionStore := newFakeSessionStore()
	logger := zap.NewNop()

	tm, err := token.NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("Gagal inisialisasi TokenManager: %v", err)
	}

	svc := NewAuthService(
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

	return svc, userRepo, roleRepo, otpRepo, rateLimiter, mailer, tm, sessionStore
}

// --- Unit Tests & Regression Tests PRD 5.7 ---

// Test Celah 1: POST /auth/register tidak dapat mem-bypass verifikasi email (status SELALU pending)
func TestAuthService_Register_Celah1_NoBypassPending(t *testing.T) {
	svc, userRepo, _, otpRepo, _, mailer, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	req := dto.RegisterRequest{
		FullName:             "Mahasiswa Baru",
		Email:                "maba@example.com",
		Phone:                "081234567890",
		Password:             "Rahasia123!",
		PasswordConfirmation: "Rahasia123!",
	}

	res, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	if res.Email != "maba@example.com" {
		t.Errorf("Email response = %s; expected maba@example.com", res.Email)
	}

	// Periksa user di database
	savedUser, err := userRepo.FindByEmail(ctx, "maba@example.com")
	if err != nil || savedUser == nil {
		t.Fatal("User harus tersimpan di database")
	}

	// REGRESI CELAH NO. 1: Status user baru WAJIB 'pending' (tidak boleh active sebelum verifikasi OTP)
	if savedUser.Status != model.UserStatusPending {
		t.Errorf("Celah No. 1 Terbuka: User status = %s; harusnya 'pending'", savedUser.Status)
	}
	if savedUser.EmailVerifiedAt != nil {
		t.Errorf("Celah No. 1 Terbuka: EmailVerifiedAt harus nil saat baru mendaftar")
	}

	// Tunggu goroutine mailer mengirim email (max 100ms)
	time.Sleep(50 * time.Millisecond)
	otpSent := mailer.GetLastOTP("maba@example.com")
	if otpSent == "" {
		t.Fatal("OTP harus dikirimkan ke email")
	}

	// REGRESI CELAH NO. 3: OTP disimpan sebagai hash SHA-256, BUKAN plaintext
	activeOTP, err := otpRepo.FindLatestActive(ctx, savedUser.ID, model.OTPPurposeEmailVerification)
	if err != nil || activeOTP == nil {
		t.Fatal("Record OTP harus ada di DB")
	}
	if activeOTP.CodeHash == otpSent {
		t.Errorf("Celah No. 3 Terbuka: OTP disimpan plaintext di database! CodeHash = %s", activeOTP.CodeHash)
	}
	if len(activeOTP.CodeHash) != 64 {
		t.Errorf("CodeHash harus berupa hex string SHA-256 sepanjang 64 karakter, didapat %d", len(activeOTP.CodeHash))
	}
}

// Test Password Policy saat Register
func TestAuthService_Register_WeakPassword(t *testing.T) {
	svc, _, _, _, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Password tanpa angka
	req := dto.RegisterRequest{
		FullName:             "Lemah Pass",
		Email:                "weak@example.com",
		Phone:                "081234567890",
		Password:             "hanyahuruf",
		PasswordConfirmation: "hanyahuruf",
	}

	_, err := svc.Register(ctx, req)
	if err == nil {
		t.Fatal("Password tanpa kombinasi angka harus ditolak")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Error harus membungkus ErrValidation, didapat: %v", err)
	}
}

// Test Register jika email sudah aktif (409 Conflict)
func TestAuthService_Register_EmailAlreadyActive(t *testing.T) {
	svc, userRepo, _, _, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	// Seed user aktif
	userRepo.Create(ctx, &model.User{
		ID:       uuid.New(),
		Email:    "aktif@example.com",
		Status:   model.UserStatusActive,
		FullName: "User Aktif",
	})

	req := dto.RegisterRequest{
		FullName:             "Coba Daftar Lagi",
		Email:                "aktif@example.com",
		Phone:                "081234567890",
		Password:             "Rahasia123!",
		PasswordConfirmation: "Rahasia123!",
	}

	_, err := svc.Register(ctx, req)
	if err == nil {
		t.Fatal("Registrasi dengan email yang sudah aktif harus gagal")
	}
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("Error = %v; expected ErrEmailTaken", err)
	}
}

// Test Celah 2: Batas Percobaan OTP (Attempts di-increment dan terkunci setelah 5x gagal)
func TestAuthService_VerifyOTP_Celah2_IncrementAttemptsAndLockout(t *testing.T) {
	svc, userRepo, _, otpRepo, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	userID := uuid.New()
	userRepo.Create(ctx, &model.User{
		ID:       userID,
		Email:    "verify@example.com",
		FullName: "Verifikasi Test",
		Status:   model.UserStatusPending,
	})

	cfgSecret := "unit-test-otp-secret-key-32bytes"
	realOTP := "654321"
	otpHash := utils.HashOTP(realOTP, cfgSecret)

	otpID := uuid.New()
	otpRepo.Create(ctx, &model.UserOTP{
		ID:          otpID,
		UserID:      userID,
		Purpose:     model.OTPPurposeEmailVerification,
		CodeHash:    otpHash,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Attempts:    0,
		MaxAttempts: 5,
	})

	// Percobaan salah 1..4 harus di-increment
	for i := 1; i <= 4; i++ {
		_, err := svc.VerifyOTP(ctx, dto.VerifyOTPRequest{
			Email: "verify@example.com",
			OTP:   "000000", // Salah
		}, "127.0.0.1", "test-agent")

		if err == nil {
			t.Fatalf("Percobaan ke-%d dengan OTP salah harus gagal", i)
		}
		if !errors.Is(err, ErrOTPInvalid) {
			t.Errorf("Percobaan ke-%d error = %v; expected ErrOTPInvalid", i, err)
		}

		// Verifikasi bahwa attempts benar-benar ter-increment di DB (CELAH NO. 2)
		activeOTP, _ := otpRepo.FindLatestActive(ctx, userID, model.OTPPurposeEmailVerification)
		if activeOTP.Attempts != int16(i) {
			t.Fatalf("Celah No. 2 Terbuka: Attempts OTP tidak di-increment! Attempts = %d, expected %d", activeOTP.Attempts, i)
		}
	}

	// Percobaan salah ke-5 (mencapai MaxAttempts)
	_, err := svc.VerifyOTP(ctx, dto.VerifyOTPRequest{
		Email: "verify@example.com",
		OTP:   "000000",
	}, "127.0.0.1", "test-agent")

	if !errors.Is(err, ErrOTPLocked) {
		t.Errorf("Percobaan ke-5 salah harus mengembalikan ErrOTPLocked, didapat: %v", err)
	}

	// Percobaan ke-6 walaupun sekarang memakai OTP yang BENAR harus tetap DITOLAK karena sudah terkunci!
	_, err = svc.VerifyOTP(ctx, dto.VerifyOTPRequest{
		Email: "verify@example.com",
		OTP:   dto.OTPCode(realOTP), // Benar, tapi sudah terkunci
	}, "127.0.0.1", "test-agent")

	if !errors.Is(err, ErrOTPLocked) {
		t.Fatalf("Celah No. 2 Terbuka: OTP yang sudah melewati batas 5 percobaan harus tetap terkunci (ErrOTPLocked), didapat: %v", err)
	}
}

// Test Verifikasi OTP Sukses (Auto-Login dan Terbit Token)
func TestAuthService_VerifyOTP_Success_AutoLogin(t *testing.T) {
	svc, userRepo, roleRepo, otpRepo, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	userID := uuid.New()
	roleCalon, _ := roleRepo.FindByCode(ctx, model.RoleCalonMahasiswa)
	emptyScope := ""

	user := &model.User{
		ID:       userID,
		Email:    "sukses@example.com",
		FullName: "Budi Santoso",
		Status:   model.UserStatusPending,
		UserRoles: []model.UserRole{
			{UserID: userID, RoleID: roleCalon.ID, Role: *roleCalon, ScopeType: &emptyScope, ScopeID: &emptyScope},
		},
	}
	userRepo.Create(ctx, user)

	realOTP := "112233"
	otpHash := utils.HashOTP(realOTP, "unit-test-otp-secret-key-32bytes")

	otpRepo.Create(ctx, &model.UserOTP{
		UserID:      userID,
		Purpose:     model.OTPPurposeEmailVerification,
		CodeHash:    otpHash,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		Attempts:    0,
		MaxAttempts: 5,
	})

	res, err := svc.VerifyOTP(ctx, dto.VerifyOTPRequest{
		Email: "sukses@example.com",
		OTP:   dto.OTPCode(realOTP),
	}, "127.0.0.1", "test-agent")

	if err != nil {
		t.Fatalf("VerifyOTP gagal: %v", err)
	}

	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatal("VerifyOTP sukses harus mengembalikan Access Token dan Refresh Token (Auto-Login)")
	}
	if len(res.User.Roles) != 1 || res.User.Roles[0] != "calon_mahasiswa" {
		t.Errorf("Roles user tidak sesuai: %v", res.User.Roles)
	}

	// Periksa status user menjadi active di DB
	savedUser, _ := userRepo.FindByEmail(ctx, "sukses@example.com")
	if savedUser.Status != model.UserStatusActive {
		t.Errorf("Status user = %s; expected 'active'", savedUser.Status)
	}
	if savedUser.EmailVerifiedAt == nil {
		t.Error("EmailVerifiedAt harus terisi")
	}
}

// Test Celah 8: Anti-Enumerasi Akun pada Login
// Login gagal saat email tidak ditemukan vs password salah mengembalikan error identik
func TestAuthService_Login_Celah8_AntiEnumerasiAkun(t *testing.T) {
	svc, userRepo, _, _, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("PasswordBenar123!"), bcrypt.MinCost)
	userRepo.Create(ctx, &model.User{
		ID:           uuid.New(),
		Email:        "ada@example.com",
		PasswordHash: string(hashedPassword),
		FullName:     "Pengguna Ada",
		Status:       model.UserStatusActive,
	})

	// Kasus A: Email TIDAK ADA di sistem
	_, errA := svc.Login(ctx, dto.LoginRequest{
		Email:    "tidakada@example.com",
		Password: "PasswordSembarang1!",
	}, "127.0.0.1", "test-agent")

	// Kasus B: Email ADA, tetapi Password SALAH
	_, errB := svc.Login(ctx, dto.LoginRequest{
		Email:    "ada@example.com",
		Password: "PasswordSalah123!",
	}, "127.0.0.1", "test-agent")

	// REGRESI CELAH NO. 8: Keduanya harus menghasilkan error yang sama persis: ErrInvalidCredentials
	if !errors.Is(errA, ErrInvalidCredentials) {
		t.Errorf("Kasus email tidak ada harus mengembalikan ErrInvalidCredentials, didapat: %v", errA)
	}
	if !errors.Is(errB, ErrInvalidCredentials) {
		t.Errorf("Kasus password salah harus mengembalikan ErrInvalidCredentials, didapat: %v", errB)
	}

	if errA.Error() != errB.Error() {
		t.Fatalf("Celah No. 8 Terbuka: Pesan error membedakan user not found vs invalid password! [%s] vs [%s]", errA.Error(), errB.Error())
	}
}

// Test Login Berhasil & Penolakan Status Pending
func TestAuthService_Login_StatusCheck(t *testing.T) {
	svc, userRepo, _, _, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.MinCost)

	// User belum verifikasi (pending)
	userRepo.Create(ctx, &model.User{
		ID:           uuid.New(),
		Email:        "pending@example.com",
		PasswordHash: string(hashedPassword),
		FullName:     "User Pending",
		Status:       model.UserStatusPending,
	})

	// User dinonaktifkan (inactive)
	userRepo.Create(ctx, &model.User{
		ID:           uuid.New(),
		Email:        "inactive@example.com",
		PasswordHash: string(hashedPassword),
		FullName:     "User Nonaktif",
		Status:       model.UserStatusInactive,
	})

	// 1. Pending login -> ErrEmailNotVerified
	_, err := svc.Login(ctx, dto.LoginRequest{
		Email:    "pending@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Errorf("Login pending user error = %v; expected ErrEmailNotVerified", err)
	}

	// 2. Inactive login -> ErrAccountInactive
	_, err = svc.Login(ctx, dto.LoginRequest{
		Email:    "inactive@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, ErrAccountInactive) {
		t.Errorf("Login inactive user error = %v; expected ErrAccountInactive", err)
	}
}

// Test GetMe
func TestAuthService_GetMe(t *testing.T) {
	svc, userRepo, roleRepo, _, _, _, _, _ := setupTestAuthService(t)
	ctx := context.Background()

	userID := uuid.New()
	roleCalon, _ := roleRepo.FindByCode(ctx, model.RoleCalonMahasiswa)
	scopeType := "jalur"
	scopeID := "prestasi"

	userRepo.Create(ctx, &model.User{
		ID:       userID,
		Email:    "me@example.com",
		FullName: "Budi Me",
		Phone:    "+6281234567890",
		Status:   model.UserStatusActive,
		UserRoles: []model.UserRole{
			{UserID: userID, RoleID: roleCalon.ID, Role: *roleCalon, ScopeType: &scopeType, ScopeID: &scopeID},
		},
	})

	me, err := svc.GetMe(ctx, userID)
	if err != nil {
		t.Fatalf("GetMe error: %v", err)
	}

	if me.Email != "me@example.com" {
		t.Errorf("GetMe email = %s; expected me@example.com", me.Email)
	}
	if len(me.Roles) != 1 || me.Roles[0] != "calon_mahasiswa" {
		t.Errorf("GetMe roles = %v", me.Roles)
	}
	if len(me.Scopes) != 1 || me.Scopes[0].ID != "prestasi" {
		t.Errorf("GetMe scopes = %v", me.Scopes)
	}
}
