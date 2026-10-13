package token

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	"go.uber.org/zap"
)

func TestTokenManager_AutoGenerateKey(t *testing.T) {
	cfg := &config.Config{
		AppEnv:            "development",
		JWTPrivateKeyPath: "", // Kosong, harus auto-generate sementara
		JWTKid:            "test-kid",
		JWTIssuer:         "pmb-auth-test",
		JWTAudience:       "pmb-api-test",
		JWTAccessTTL:      15 * time.Minute,
		JWTRefreshTTL:     7 * 24 * time.Hour,
	}

	logger := zap.NewNop()
	tm, err := NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("Gagal inisialisasi TokenManager dengan auto-generate key: %v", err)
	}

	if tm.PublicKey() == nil {
		t.Fatal("PublicKey tidak boleh nil")
	}

	jwks := tm.GetJWKS()
	if len(jwks.Keys) != 1 {
		t.Fatalf("JWKS keys harus memiliki panjang 1, didapat %d", len(jwks.Keys))
	}
	if jwks.Keys[0].Kid != "test-kid" {
		t.Errorf("JWKS kid = %s; expected %s", jwks.Keys[0].Kid, "test-kid")
	}
	if jwks.Keys[0].Alg != "RS256" {
		t.Errorf("JWKS alg = %s; expected %s", jwks.Keys[0].Alg, "RS256")
	}
}

func TestTokenManager_LoadFromFile(t *testing.T) {
	// Jalurnya mengarah ke certs/jwt_private.pem yang digenerate di root project
	privPath, _ := filepath.Abs("../../../../certs/jwt_private.pem")

	cfg := &config.Config{
		AppEnv:            "development",
		JWTPrivateKeyPath: privPath,
		JWTKid:            "pmb-auth-file",
		JWTIssuer:         "pmb-auth-test",
		JWTAudience:       "pmb-api-test",
		JWTAccessTTL:      15 * time.Minute,
		JWTRefreshTTL:     7 * 24 * time.Hour,
	}

	logger := zap.NewNop()
	tm, err := NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("Gagal inisialisasi TokenManager dari file: %v", err)
	}

	if tm.PublicKey() == nil {
		t.Fatal("PublicKey tidak boleh nil setelah memuat dari file")
	}
}

func TestTokenManager_ProductionRequiresKeyPath(t *testing.T) {
	cfg := &config.Config{
		AppEnv:            "production",
		JWTPrivateKeyPath: "", // Di production wajib ada
	}

	logger := zap.NewNop()
	_, err := NewTokenManager(cfg, logger)
	if err == nil {
		t.Fatal("NewTokenManager di production tanpa JWT_PRIVATE_KEY_PATH harus menghasilkan error")
	}
}

func TestTokenManager_GenerateAndValidateTokens(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "test",
		JWTKid:        "test-kid",
		JWTIssuer:     "pmb-auth-test",
		JWTAudience:   "pmb-api-test",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	logger := zap.NewNop()
	tm, err := NewTokenManager(cfg, logger)
	if err != nil {
		t.Fatalf("Gagal membuat TokenManager: %v", err)
	}

	userID := uuid.New()
	email := "calon@example.com"
	name := "Budi Santoso"
	roles := []string{"calon_mahasiswa"}
	scopes := []ScopeClaim{
		{Role: "calon_mahasiswa", Type: "jalur", ID: "reguler"},
	}
	sessionID := uuid.NewString()
	permissions := []string{"admission.document.verify", "dashboard.overview.read"}

	// 1. Generate Access Token
	accessTokenStr, accessJTI, accessExp, err := tm.GenerateAccessToken(userID, email, name, roles, scopes, permissions, sessionID)
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	if accessTokenStr == "" || accessJTI == "" {
		t.Fatal("Access token string dan JTI tidak boleh kosong")
	}
	if accessExp.Before(time.Now()) {
		t.Fatal("Access token expiration tidak valid")
	}

	// 2. Validate Access Token
	accessClaims, err := tm.ValidateAccessToken(accessTokenStr)
	if err != nil {
		t.Fatalf("ValidateAccessToken gagal: %v", err)
	}
	if accessClaims.Typ != "access" {
		t.Errorf("Claim typ = %s; expected 'access'", accessClaims.Typ)
	}
	if len(accessClaims.Permissions) != 2 || accessClaims.Permissions[0] != "admission.document.verify" {
		t.Errorf("Claim permissions tidak sesuai: %v", accessClaims.Permissions)
	}
	if accessClaims.Email != email {
		t.Errorf("Claim email = %s; expected %s", accessClaims.Email, email)
	}
	if accessClaims.Subject != userID.String() {
		t.Errorf("Claim sub = %s; expected %s", accessClaims.Subject, userID.String())
	}
	if len(accessClaims.Roles) != 1 || accessClaims.Roles[0] != "calon_mahasiswa" {
		t.Errorf("Claim roles tidak sesuai: %v", accessClaims.Roles)
	}
	if len(accessClaims.Scopes) != 1 || accessClaims.Scopes[0].ID != "reguler" {
		t.Errorf("Claim scopes tidak sesuai: %v", accessClaims.Scopes)
	}
	if accessClaims.Sid != sessionID {
		t.Errorf("Claim sid = %s; expected %s", accessClaims.Sid, sessionID)
	}

	// 3. Generate Refresh Token
	refreshTokenStr, refreshJTI, refreshExp, err := tm.GenerateRefreshToken(userID, sessionID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if refreshTokenStr == "" || refreshJTI == "" {
		t.Fatal("Refresh token string dan JTI tidak boleh kosong")
	}
	if refreshExp.Before(time.Now()) {
		t.Fatal("Refresh token expiration tidak valid")
	}

	// 4. Validate Refresh Token
	refreshClaims, err := tm.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		t.Fatalf("ValidateRefreshToken gagal: %v", err)
	}
	if refreshClaims.Typ != "refresh" {
		t.Errorf("Claim typ = %s; expected 'refresh'", refreshClaims.Typ)
	}
	if refreshClaims.Subject != userID.String() {
		t.Errorf("Claim sub = %s; expected %s", refreshClaims.Subject, userID.String())
	}
	if refreshClaims.Sid != sessionID {
		t.Errorf("Claim sid = %s; expected %s", refreshClaims.Sid, sessionID)
	}

	// 5. REGRESI CELAH KEAMANAN NO. 7:
	// Access token dan refresh token TIDAK boleh bisa saling menggantikan!
	// Refresh token DITOLAK jika divalidasi sebagai access token:
	_, err = tm.ValidateAccessToken(refreshTokenStr)
	if err == nil {
		t.Fatal("Celah No. 7: Refresh token TIDAK boleh diterima sebagai access token!")
	}
	if err.Error() != "tipe token bukan access token" {
		t.Logf("Error saat memvalidasi refresh token sebagai access token: %v", err)
	}

	// Access token DITOLAK jika divalidasi sebagai refresh token:
	_, err = tm.ValidateRefreshToken(accessTokenStr)
	if err == nil {
		t.Fatal("Celah No. 7: Access token TIDAK boleh diterima sebagai refresh token!")
	}
	if err.Error() != "tipe token bukan refresh token" {
		t.Logf("Error saat memvalidasi access token sebagai refresh token: %v", err)
	}
}
