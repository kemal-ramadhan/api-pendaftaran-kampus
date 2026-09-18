package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/config"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/health"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/middleware"
	"github.com/pkm-asia/pmb-backend/services/api-gateway/internal/proxy"
	"go.uber.org/zap"
)

type mockJWKSClient struct {
	keys map[string]*rsa.PublicKey
}

func (m *mockJWKSClient) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := m.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("kid not found: %s", kid)
}

func (m *mockJWKSClient) Refresh(ctx context.Context) error {
	return nil
}

func (m *mockJWKSClient) SetStaticKey(kid string, pubKey *rsa.PublicKey) {
	m.keys[kid] = pubKey
}

func setupTestGateway(
	t *testing.T,
	upstreamServer *httptest.Server,
	privKey *rsa.PrivateKey,
	kid string,
) (*httptest.Server, *config.Config, *mockJWKSClient) {
	logger := zap.NewNop()

	upstreamURL, _ := url.Parse(upstreamServer.URL)

	cfg := &config.Config{
		AppEnv:              "test",
		AppPort:             "8000",
		AuthServiceURL:      upstreamURL.String(),
		JWKSURL:             upstreamURL.String() + "/.well-known/jwks.json",
		JWTIssuer:           "pmb-auth-test",
		JWTAudience:         "pmb-api-test",
		GatewaySharedSecret: "test-gateway-secret-32-chars-long",
		CORSAllowedOrigins:  []string{"http://localhost:5173"},
		RateLimitGlobalReq:  1000,
		RateLimitGlobalWin:  time.Minute,
		Routes: &config.RoutesConfig{
			Services: map[string]config.ServiceConfig{
				"auth": {
					URL:     upstreamURL.String(),
					Timeout: 2 * time.Second,
				},
			},
			Routes: []config.RouteConfig{
				{
					Prefix:  "/api/v1/auth/.well-known",
					Service: "auth",
					Rewrite: "/.well-known",
					Auth:    "public",
				},
				{
					Prefix:  "/api/v1/auth/admin",
					Service: "auth",
					Rewrite: "/auth/admin",
					Auth:    "required",
					Roles:   []string{"superadmin"},
				},
				{
					Prefix:  "/api/v1/auth",
					Service: "auth",
					Rewrite: "/auth",
					Auth:    "required",
					PublicPaths: []string{
						"POST /register",
						"POST /verify",
						"GET /health",
						"POST /login",
						"POST /refresh",
						"POST /forgot-password",
						"POST /reset-password",
					},
				},
			},
		},
	}

	jwksMock := &mockJWKSClient{
		keys: map[string]*rsa.PublicKey{
			kid: &privKey.PublicKey,
		},
	}

	proxyHandler, err := proxy.NewProxyHandler(cfg.Routes, logger)
	if err != nil {
		t.Fatalf("NewProxyHandler error: %v", err)
	}

	healthHandler := health.NewHealthHandler(cfg.Routes)
	engine := SetupGatewayEngine(cfg, jwksMock, nil, proxyHandler, healthHandler, logger)

	gwServer := httptest.NewServer(engine)
	return gwServer, cfg, jwksMock
}

func generateTestToken(
	privKey *rsa.PrivateKey,
	kid string,
	userID string,
	email string,
	roles []string,
	typ string,
	exp time.Duration,
) string {
	claims := middleware.GatewayClaims{
		Typ:   typ,
		Email: email,
		Name:  "Test User",
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    "pmb-auth-test",
			Audience:  jwt.ClaimStrings{"pmb-api-test"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenStr, _ := token.SignedString(privKey)
	return tokenStr
}

// 1. Test Pencocokan Route & Public Paths
func TestGateway_RouteMatching(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "key-1"

	// Mock Upstream Server yang memvalidasi path yang diterima
	var lastReceivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReceivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer upstream.Close()

	gw, _, _ := setupTestGateway(t, upstream, privKey, kid)
	defer gw.Close()

	// A. Public path POST /api/v1/auth/login -> lolos tanpa token, rewritten ke /auth/login
	resp1, err := http.Post(gw.URL+"/api/v1/auth/login", "application/json", nil)
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/v1/auth/login status = %d; expected 200", resp1.StatusCode)
	}
	if lastReceivedPath != "/auth/login" {
		t.Errorf("Upstream path = %s; expected /auth/login", lastReceivedPath)
	}

	// B. Route terproteksi GET /api/v1/auth/me tanpa token -> 401 Unauthorized
	resp2, _ := http.Get(gw.URL + "/api/v1/auth/me")
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/auth/me tanpa token status = %d; expected 401", resp2.StatusCode)
	}

	// C. Route tidak dikenal -> 404 Not Found
	resp3, _ := http.Get(gw.URL + "/api/v1/unknown")
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("GET /api/v1/unknown status = %d; expected 404", resp3.StatusCode)
	}
}

// 2. Test Blokir Path Sensitif Hilir (/internal/*, /metrics, /docs)
func TestGateway_BlockProtectedPaths(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _, _ := setupTestGateway(t, upstream, privKey, "k1")
	defer gw.Close()

	blockedPaths := []string{
		"/api/v1/auth/internal/users/123",
		"/internal/secret",
		"/metrics",
		"/api/v1/auth/docs/index.html",
	}

	for _, p := range blockedPaths {
		resp, _ := http.Get(gw.URL + p)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Path %s harus diblokir (404), didapat status: %d", p, resp.StatusCode)
		}
	}
}

// 3. Test CORS Preflight
func TestGateway_CORS_Preflight(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _, _ := setupTestGateway(t, upstream, privKey, "k1")
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodOptions, gw.URL+"/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("CORS request error: %v", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("CORS preflight status = %d; expected 204 No Content", resp.StatusCode)
	}
	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %s; expected http://localhost:5173", allowOrigin)
	}
	if resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials harus true")
	}
}

// 4. Test Validasi JWT & RBAC Role Check
func TestGateway_JWTValidation_And_RoleCheck(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "k1"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer upstream.Close()

	gw, _, _ := setupTestGateway(t, upstream, privKey, kid)
	defer gw.Close()

	userID := uuid.NewString()

	// A. Token kedaluwarsa -> 401 TOKEN_EXPIRED
	expiredToken := generateTestToken(privKey, kid, userID, "user@test.com", []string{"calon_mahasiswa"}, "access", -10*time.Minute)
	reqExp, _ := http.NewRequest(http.MethodGet, gw.URL+"/api/v1/auth/me", nil)
	reqExp.Header.Set("Authorization", "Bearer "+expiredToken)
	respExp, _ := http.DefaultClient.Do(reqExp)
	if respExp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expired token status = %d; expected 401", respExp.StatusCode)
	}

	// B. Celah 7: Refresh token dipakai sebagai access token -> 401 TOKEN_INVALID
	refreshToken := generateTestToken(privKey, kid, userID, "user@test.com", []string{"calon_mahasiswa"}, "refresh", 15*time.Minute)
	reqRef, _ := http.NewRequest(http.MethodGet, gw.URL+"/api/v1/auth/me", nil)
	reqRef.Header.Set("Authorization", "Bearer "+refreshToken)
	respRef, _ := http.DefaultClient.Do(reqRef)
	if respRef.StatusCode != http.StatusUnauthorized {
		t.Errorf("Refresh token as access token status = %d; expected 401", respRef.StatusCode)
	}

	// C. Role check: endpoint admin mensyaratkan role superadmin
	calonToken := generateTestToken(privKey, kid, userID, "calon@test.com", []string{"calon_mahasiswa"}, "access", 15*time.Minute)
	reqAdminCalon, _ := http.NewRequest(http.MethodGet, gw.URL+"/api/v1/auth/admin/users", nil)
	reqAdminCalon.Header.Set("Authorization", "Bearer "+calonToken)
	respAdminCalon, _ := http.DefaultClient.Do(reqAdminCalon)
	if respAdminCalon.StatusCode != http.StatusForbidden {
		t.Errorf("Calon mahasiswa mengakses admin route status = %d; expected 403 Forbidden", respAdminCalon.StatusCode)
	}

	// D. Superadmin mengakses endpoint admin -> 200 OK
	superToken := generateTestToken(privKey, kid, userID, "super@test.com", []string{"superadmin"}, "access", 15*time.Minute)
	reqAdminSuper, _ := http.NewRequest(http.MethodGet, gw.URL+"/api/v1/auth/admin/users", nil)
	reqAdminSuper.Header.Set("Authorization", "Bearer "+superToken)
	respAdminSuper, _ := http.DefaultClient.Do(reqAdminSuper)
	if respAdminSuper.StatusCode != http.StatusOK {
		t.Errorf("Superadmin mengakses admin route status = %d; expected 200 OK", respAdminSuper.StatusCode)
	}
}

// 5. Test Anti-Spoofing Header & Injeksi HMAC Gateway Signature
func TestGateway_AntiSpoofing_And_HMACInjection(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "k1"

	var receivedHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer upstream.Close()

	gw, cfg, _ := setupTestGateway(t, upstream, privKey, kid)
	defer gw.Close()

	realUserID := uuid.NewString()
	tokenStr := generateTestToken(privKey, kid, realUserID, "real@univ.ac.id", []string{"keuangan"}, "access", 15*time.Minute)

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	// Spoofed headers yang dikirim penyerang dari klien:
	req.Header.Set("X-User-ID", "fake-admin-uuid")
	req.Header.Set("X-User-Roles", "superadmin")
	req.Header.Set("X-Gateway-Signature", "fake-signature")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Request error status = %d", resp.StatusCode)
	}

	// 1. Header palsu harus DIBERSIHKAN
	if receivedHeaders.Get("X-User-ID") == "fake-admin-uuid" {
		t.Fatal("Celah Spoofing Terbuka: X-User-ID palsu dari klien lolos ke upstream!")
	}
	if receivedHeaders.Get("X-User-Roles") == "superadmin" {
		t.Fatal("Celah Spoofing Terbuka: X-User-Roles palsu dari klien lolos ke upstream!")
	}

	// 2. Header asli terverifikasi harus DIINJEKSI
	if receivedHeaders.Get("X-User-ID") != realUserID {
		t.Errorf("X-User-ID di upstream = %s; expected %s", receivedHeaders.Get("X-User-ID"), realUserID)
	}
	if receivedHeaders.Get("X-User-Roles") != "keuangan" {
		t.Errorf("X-User-Roles di upstream = %s; expected keuangan", receivedHeaders.Get("X-User-Roles"))
	}

	// 3. Signature HMAC harus VALID
	ts := receivedHeaders.Get("X-Gateway-Timestamp")
	sig := receivedHeaders.Get("X-Gateway-Signature")
	reqID := receivedHeaders.Get("X-Request-ID")
	if !gatewayauth.VerifySignature(cfg.GatewaySharedSecret, reqID, realUserID, ts, sig, 60*time.Second) {
		t.Fatal("Signature HMAC gateway yang diinjeksi ke upstream tidak valid!")
	}
}

// 6. Test Upstream Error 502 Bad Gateway
func TestGateway_UpstreamBadGateway(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Upstream server langsung ditutup untuk mensimulasikan server mati
	deadUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	deadURL := deadUpstream.URL
	deadUpstream.Close() // Matikan segera

	logger := zap.NewNop()
	cfg := &config.Config{
		AppEnv:              "test",
		AppPort:             "8000",
		GatewaySharedSecret: "test-secret",
		Routes: &config.RoutesConfig{
			Services: map[string]config.ServiceConfig{
				"auth": {URL: deadURL, Timeout: 1 * time.Second},
			},
			Routes: []config.RouteConfig{
				{Prefix: "/api/v1/auth", Service: "auth", Rewrite: "/auth", Auth: "public"},
			},
		},
	}

	proxyHandler, _ := proxy.NewProxyHandler(cfg.Routes, logger)
	healthHandler := health.NewHealthHandler(cfg.Routes)
	jwksMock := &mockJWKSClient{keys: map[string]*rsa.PublicKey{"k1": &privKey.PublicKey}}
	engine := SetupGatewayEngine(cfg, jwksMock, nil, proxyHandler, healthHandler, logger)

	gw := httptest.NewServer(engine)
	defer gw.Close()

	resp, _ := http.Get(gw.URL + "/api/v1/auth/health")
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("Upstream mati status = %d; expected 502 Bad Gateway", resp.StatusCode)
	}

	var errResp response.ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.ErrorCode != response.ErrCodeUpstreamUnavailable {
		t.Errorf("ErrorCode = %s; expected %s", errResp.ErrorCode, response.ErrCodeUpstreamUnavailable)
	}
}
