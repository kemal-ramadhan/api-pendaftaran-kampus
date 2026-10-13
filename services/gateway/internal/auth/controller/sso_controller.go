package controller

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/config"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/dto"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/repository"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/service"
	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/token"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type SSOController struct {
	cfg          *config.Config
	authService  service.AuthService
	serviceRepo  repository.ServiceAccountRepository
	userRepo     repository.UserRepository
	tokenManager token.TokenManager
	sessionStore token.SessionStore
	rdb          *redis.Client
	log          *zap.Logger
}

func NewSSOController(
	cfg *config.Config,
	authService service.AuthService,
	serviceRepo repository.ServiceAccountRepository,
	userRepo repository.UserRepository,
	tokenManager token.TokenManager,
	sessionStore token.SessionStore,
	rdb *redis.Client,
	log *zap.Logger,
) *SSOController {
	return &SSOController{
		cfg:          cfg,
		authService:  authService,
		serviceRepo:  serviceRepo,
		userRepo:     userRepo,
		tokenManager: tokenManager,
		sessionStore: sessionStore,
		rdb:          rdb,
		log:          log,
	}
}

// GetOpenIDConfiguration mengembalikan OIDC discovery document (RFC 8414)
func (ctl *SSOController) GetOpenIDConfiguration(c *gin.Context) {
	baseURL := strings.TrimRight(ctl.cfg.FrontendURL, "/")
	issuer := ctl.cfg.JWTIssuer
	if issuer == "" {
		issuer = baseURL
	}

	c.JSON(http.StatusOK, gin.H{
		"issuer":                                issuer,
		"authorization_endpoint":                baseURL + "/oauth2/authorize",
		"token_endpoint":                        baseURL + "/oauth2/token",
		"userinfo_endpoint":                     baseURL + "/oauth2/userinfo",
		"jwks_uri":                              baseURL + "/.well-known/jwks.json",
		"revocation_endpoint":                   baseURL + "/oauth2/revoke",
		"end_session_endpoint":                  baseURL + "/oauth2/logout",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "name", "email", "email_verified", "roles", "svc"},
		"code_challenge_methods_supported":      []string{"S256"},
	})
}

// Authorize menangani GET /oauth2/authorize dengan PKCE S256
func (ctl *SSOController) Authorize(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	responseType := c.Query("response_type")
	state := c.Query("state")
	nonce := c.Query("nonce")
	codeChallenge := c.Query("code_challenge")
	codeChallengeMethod := c.Query("code_challenge_method")

	if clientID == "" || redirectURI == "" || responseType != "code" {
		c.String(http.StatusBadRequest, "Permintaan OAuth2 tidak valid: client_id, redirect_uri, dan response_type=code wajib diisi")
		return
	}

	if codeChallenge == "" || codeChallengeMethod != "S256" {
		c.String(http.StatusBadRequest, "PKCE S256 wajib disertakan (code_challenge dan code_challenge_method=S256)")
		return
	}

	sa, err := ctl.serviceRepo.GetServiceAccountByCode(c.Request.Context(), clientID)
	if err != nil || sa == nil || !sa.IsActive {
		c.String(http.StatusUnauthorized, "Service client tidak terdaftar atau tidak aktif")
		return
	}

	// Cek redirect_uri terdaftar
	if sa.RedirectURIs != "" {
		allowedURIs := strings.Split(sa.RedirectURIs, ",")
		valid := false
		for _, u := range allowedURIs {
			if strings.TrimSpace(u) == redirectURI || strings.HasPrefix(redirectURI, "http://localhost") {
				valid = true
				break
			}
		}
		if !valid && !strings.HasPrefix(redirectURI, "http://localhost") {
			c.String(http.StatusBadRequest, "redirect_uri tidak terdaftar untuk client ini")
			return
		}
	}

	// Cek session login SSO via cookie atau header
	sessionCookie, err := c.Cookie("sso_session")
	var userIDStr string
	if err == nil && sessionCookie != "" && ctl.rdb != nil {
		userIDStr, _ = ctl.rdb.Get(c.Request.Context(), "sso:session:"+sessionCookie).Result()
	}

	if userIDStr == "" {
		// Belum login, arahkan ke /sso/login
		returnTo := c.Request.RequestURI
		c.Redirect(http.StatusFound, "/sso/login?continue="+base64.URLEncoding.EncodeToString([]byte(returnTo)))
		return
	}

	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/sso/login")
		return
	}

	// Cek hak akses user ke service account
	user, err := ctl.userRepo.FindByID(c.Request.Context(), uid)
	if err != nil || user == nil {
		c.Redirect(http.StatusFound, "/sso/login")
		return
	}

	isSuperadmin := false
	for _, ur := range user.UserRoles {
		if ur.Role.Code == model.RoleSuperadmin {
			isSuperadmin = true
			break
		}
	}

	if !isSuperadmin {
		ua, err := ctl.serviceRepo.GetUserAccount(c.Request.Context(), uid, sa.ID)
		if err != nil || ua == nil || ua.Status != model.UserAccountStatusActive {
			c.String(http.StatusForbidden, "Anda tidak memiliki akses ke layanan ini")
			return
		}
	}

	// Buat Authorization Code (berlaku 60 detik)
	authCode := uuid.NewString()
	authCodeData := fmt.Sprintf("%s|%s|%s|%s|%s", uid.String(), clientID, redirectURI, codeChallenge, nonce)
	if ctl.rdb != nil {
		ctl.rdb.Set(c.Request.Context(), "sso:code:"+authCode, authCodeData, 60*time.Second)
	}

	delim := "?"
	if strings.Contains(redirectURI, "?") {
		delim = "&"
	}
	targetURL := fmt.Sprintf("%s%scode=%s", redirectURI, delim, authCode)
	if state != "" {
		targetURL += "&state=" + state
	}

	c.Redirect(http.StatusFound, targetURL)
}

// Token menangani POST /oauth2/token
func (ctl *SSOController) Token(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	if grantType == "" {
		var body struct {
			GrantType    string `json:"grant_type"`
			Code         string `json:"code"`
			CodeVerifier string `json:"code_verifier"`
			ClientID     string `json:"client_id"`
			RedirectURI  string `json:"redirect_uri"`
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			grantType = body.GrantType
			c.Set("json_code", body.Code)
			c.Set("json_verifier", body.CodeVerifier)
			c.Set("json_client_id", body.ClientID)
			c.Set("json_redirect_uri", body.RedirectURI)
			c.Set("json_refresh_token", body.RefreshToken)
		}
	}

	if grantType == "authorization_code" {
		code := c.PostForm("code")
		if code == "" {
			if v, ok := c.Get("json_code"); ok {
				code = v.(string)
			}
		}
		codeVerifier := c.PostForm("code_verifier")
		if codeVerifier == "" {
			if v, ok := c.Get("json_verifier"); ok {
				codeVerifier = v.(string)
			}
		}
		clientID := c.PostForm("client_id")
		if clientID == "" {
			if v, ok := c.Get("json_client_id"); ok {
				clientID = v.(string)
			}
		}

		if code == "" || codeVerifier == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "code dan code_verifier wajib diisi"})
			return
		}

		var codeData string
		var err error
		if ctl.rdb != nil {
			codeData, err = ctl.rdb.Get(c.Request.Context(), "sso:code:"+code).Result()
			ctl.rdb.Del(c.Request.Context(), "sso:code:"+code) // Sekali pakai
		}
		if err != nil || codeData == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "Authorization code tidak valid atau telah kedaluwarsa"})
			return
		}

		parts := strings.Split(codeData, "|")
		if len(parts) < 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "Data code rusak"})
			return
		}
		userIDStr, storedClientID, _, storedChallenge := parts[0], parts[1], parts[2], parts[3]
		nonce := ""
		if len(parts) >= 5 {
			nonce = parts[4]
		}

		if clientID != "" && clientID != storedClientID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "client_id tidak cocok"})
			return
		}

		// Verifikasi PKCE S256: BASE64URL-ENCODE(SHA256(ASCII(code_verifier))) == code_challenge
		h := sha256.Sum256([]byte(codeVerifier))
		computedChallenge := base64.RawURLEncoding.EncodeToString(h[:])
		if computedChallenge != storedChallenge {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "code_verifier PKCE tidak cocok"})
			return
		}

		uid, _ := uuid.Parse(userIDStr)
		user, err := ctl.userRepo.FindByID(c.Request.Context(), uid)
		if err != nil || user == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "Pengguna tidak ditemukan"})
			return
		}

		sa, _ := ctl.serviceRepo.GetServiceAccountByCode(c.Request.Context(), storedClientID)
		audience := "pmb-api"
		if sa != nil && sa.Audience != "" {
			audience = sa.Audience
		}

		roles := make([]string, 0)
		for _, ur := range user.UserRoles {
			if ur.Role.Code != "" {
				roles = append(roles, ur.Role.Code)
			}
		}

		sessionID := uuid.NewString()
		accessToken, _, exp, err := ctl.tokenManager.GenerateAccessTokenWithServices(
			user.ID,
			user.Email,
			user.FullName,
			roles,
			nil,
			nil,
			sessionID,
			[]string{storedClientID},
			audience,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
			return
		}

		refreshToken, refreshJTI, _, _ := ctl.tokenManager.GenerateRefreshToken(user.ID, sessionID)
		_ = ctl.sessionStore.SaveSession(c.Request.Context(), user.ID, sessionID, refreshJTI, ctl.cfg.JWTRefreshTTL)

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"expires_in":    int64(time.Until(exp).Seconds()),
			"refresh_token": refreshToken,
			"id_token":      accessToken, // ID token kompatibel
			"scope":         "openid profile email",
			"nonce":         nonce,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type", "error_description": "Grant type tidak didukung"})
}

// UserInfo menangani GET /oauth2/userinfo
func (ctl *SSOController) UserInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := ctl.tokenManager.ValidateAccessToken(tokenStr)
	if err != nil || claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sub":                claims.Subject,
		"name":               claims.Name,
		"email":              claims.Email,
		"email_verified":     claims.EmailVerified,
		"preferred_username": claims.PreferredUsername,
		"roles":              claims.Roles,
		"svc":                claims.Services,
	})
}

// Revoke menangani POST /oauth2/revoke
func (ctl *SSOController) Revoke(c *gin.Context) {
	tokenStr := c.PostForm("token")
	if tokenStr == "" {
		var body struct {
			Token string `json:"token"`
		}
		_ = c.ShouldBindJSON(&body)
		tokenStr = body.Token
	}

	if tokenStr != "" {
		refClaims, err := ctl.tokenManager.ValidateRefreshToken(tokenStr)
		if err == nil && refClaims != nil {
			uid, _ := uuid.Parse(refClaims.Subject)
			_ = ctl.sessionStore.RevokeSession(c.Request.Context(), uid, refClaims.Sid)
		}
	}
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

// Logout menangani GET/POST /oauth2/logout
func (ctl *SSOController) Logout(c *gin.Context) {
	sessionCookie, err := c.Cookie("sso_session")
	if err == nil && sessionCookie != "" && ctl.rdb != nil {
		ctl.rdb.Del(c.Request.Context(), "sso:session:"+sessionCookie)
	}

	c.SetCookie("sso_session", "", -1, "/", "", false, true)
	postLogoutRedirect := c.Query("post_logout_redirect_uri")
	if postLogoutRedirect != "" {
		c.Redirect(http.StatusFound, postLogoutRedirect)
		return
	}
	response.Success(c, http.StatusOK, "Logout SSO berhasil", nil)
}

// ShowLoginPage merender halaman form login SSO sementara (HTML)
func (ctl *SSOController) ShowLoginPage(c *gin.Context) {
	continueParam := c.Query("continue")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Masuk - Single Sign-On Kampus</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f3f4f6; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
        .card { background: white; padding: 2rem; border-radius: 0.5rem; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1); width: 100%%; max-width: 400px; }
        h1 { font-size: 1.25rem; font-weight: bold; margin-bottom: 0.5rem; color: #111827; }
        p { font-size: 0.875rem; color: #6b7280; margin-bottom: 1.5rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; font-size: 0.875rem; font-weight: 500; color: #374151; margin-bottom: 0.25rem; }
        input[type="email"], input[type="password"] { width: 100%%; padding: 0.5rem 0.75rem; border: 1px solid #d1d5db; border-radius: 0.375rem; box-sizing: border-box; }
        button { width: 100%%; background: #2563eb; color: white; padding: 0.625rem; border: none; border-radius: 0.375rem; font-weight: 500; cursor: pointer; margin-top: 1rem; }
        button:hover { background: #1d4ed8; }
    </style>
</head>
<body>
    <div class="card">
        <h1>Portal SSO Kampus</h1>
        <p>Silakan masuk menggunakan akun universitas Anda.</p>
        <form method="POST" action="/sso/login">
            <input type="hidden" name="continue" value="%s">
            <div class="form-group">
                <label for="email">Alamat Email</label>
                <input type="email" id="email" name="email" required autocomplete="email">
            </div>
            <div class="form-group">
                <label for="password">Kata Sandi</label>
                <input type="password" id="password" name="password" required autocomplete="current-password">
            </div>
            <button type="submit">Masuk</button>
        </form>
    </div>
</body>
</html>`, continueParam)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// ProcessLogin menangani form submission login SSO
func (ctl *SSOController) ProcessLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	continueParam := c.PostForm("continue")

	req := dto.LoginRequest{
		Email:    email,
		Password: password,
	}

	authData, err := ctl.authService.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.String(http.StatusUnauthorized, "Email atau kata sandi tidak valid")
		return
	}

	// Buat cookie sesi SSO
	ssoSessionID := uuid.NewString()
	if ctl.rdb != nil {
		ctl.rdb.Set(c.Request.Context(), "sso:session:"+ssoSessionID, authData.User.ID.String(), 8*time.Hour)
	}
	c.SetCookie("sso_session", ssoSessionID, 8*3600, "/", "", false, true)

	if continueParam != "" {
		decoded, err := base64.URLEncoding.DecodeString(continueParam)
		if err == nil && string(decoded) != "" {
			c.Redirect(http.StatusFound, string(decoded))
			return
		}
	}

	c.Redirect(http.StatusFound, "/auth/me")
}
