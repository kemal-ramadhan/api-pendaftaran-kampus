package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppEnv              string
	AppPort             string
	RoutesConfigPath    string
	AuthServiceURL      string
	JWKSURL             string
	JWTIssuer           string
	JWTAudience         string
	RedisAddr           string
	RedisPassword       string
	GatewaySharedSecret string
	CORSAllowedOrigins  []string
	RateLimitGlobalReq  int
	RateLimitGlobalWin  time.Duration
	TrustedProxies      []string
	Routes              *RoutesConfig
}

type ServiceConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
	MaxBody string        `yaml:"max_body"`
}

type RateLimitConfig struct {
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

type RouteConfig struct {
	Prefix      string           `yaml:"prefix"`
	Service     string           `yaml:"service"`
	Rewrite     string           `yaml:"rewrite"`
	Auth        string           `yaml:"auth"` // "public", "required"
	Roles       []string         `yaml:"roles"`
	PublicPaths []string         `yaml:"public_paths"`
	RateLimit   *RateLimitConfig `yaml:"rate_limit"`
	MaxBody     string           `yaml:"max_body"`
}

type RoutesConfig struct {
	Services map[string]ServiceConfig `yaml:"services"`
	Routes   []RouteConfig            `yaml:"routes"`
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func parseRateLimit(s string) (int, time.Duration) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 120, time.Minute
	}
	req, err := strconv.Atoi(parts[0])
	if err != nil {
		req = 120
	}
	dur, err := time.ParseDuration(parts[1])
	if err != nil {
		dur = time.Minute
	}
	return req, dur
}

func LoadConfig() (*Config, error) {
	appEnv := getEnv("APP_ENV", "development")
	appPort := getEnv("APP_PORT", "8000")
	routesPath := getEnv("ROUTES_CONFIG", "config/routes.yaml")
	authURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8080")
	jwksURL := getEnv("JWKS_URL", authURL+"/.well-known/jwks.json")
	jwtIssuer := getEnv("JWT_ISSUER", "pmb-auth-service")
	jwtAudience := getEnv("JWT_AUDIENCE", "pmb-api")
	redisAddr := getEnv("REDIS_ADDR", "redis:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	gatewaySecret := getEnv("GATEWAY_SHARED_SECRET", "")
	corsRaw := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	rateLimitRaw := getEnv("RATE_LIMIT_GLOBAL", "120/1m")
	trustedProxiesRaw := getEnv("TRUSTED_PROXIES", "")

	reqLimit, winLimit := parseRateLimit(rateLimitRaw)

	corsOrigins := make([]string, 0)
	for _, o := range strings.Split(corsRaw, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			corsOrigins = append(corsOrigins, trimmed)
		}
	}

	trustedProxies := make([]string, 0)
	if trustedProxiesRaw != "" {
		for _, p := range strings.Split(trustedProxiesRaw, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				trustedProxies = append(trustedProxies, trimmed)
			}
		}
	}

	// Cari file routes.yaml
	resolvedRoutesPath := routesPath
	if _, err := os.Stat(resolvedRoutesPath); os.IsNotExist(err) {
		// Coba path alternatif untuk dev lokal
		candidates := []string{
			"config/routes.yaml",
			"../config/routes.yaml",
			"services/api-gateway/config/routes.yaml",
			filepath.Join("services", "api-gateway", "config", "routes.yaml"),
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				resolvedRoutesPath = cand
				break
			}
		}
	}

	routesConfig, err := LoadRoutesConfig(resolvedRoutesPath)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat konfigurasi routes: %w", err)
	}

	cfg := &Config{
		AppEnv:              appEnv,
		AppPort:             appPort,
		RoutesConfigPath:    resolvedRoutesPath,
		AuthServiceURL:      authURL,
		JWKSURL:             jwksURL,
		JWTIssuer:           jwtIssuer,
		JWTAudience:         jwtAudience,
		RedisAddr:           redisAddr,
		RedisPassword:       redisPassword,
		GatewaySharedSecret: gatewaySecret,
		CORSAllowedOrigins:  corsOrigins,
		RateLimitGlobalReq:  reqLimit,
		RateLimitGlobalWin:  winLimit,
		TrustedProxies:      trustedProxies,
		Routes:              routesConfig,
	}

	return cfg, nil
}

// LoadRoutesConfig membaca routes.yaml dan melakukan substitusi env var
func LoadRoutesConfig(filePath string) (*RoutesConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Substitusi env var format ${VAR_NAME}
	re := regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)
	substituted := re.ReplaceAllStringFunc(string(data), func(m string) string {
		varName := m[2 : len(m)-1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		// Default fallback jika AUTH_SERVICE_URL / OCR_SERVICE_URL tidak ada di env
		if varName == "AUTH_SERVICE_URL" {
			return "http://auth-service:8080"
		}
		if varName == "OCR_SERVICE_URL" {
			return "http://ocr-service:8088"
		}
		return m
	})

	var rc RoutesConfig
	if err := yaml.Unmarshal([]byte(substituted), &rc); err != nil {
		return nil, err
	}

	// Validasi routes: setiap service harus terdaftar
	for i, r := range rc.Routes {
		svc, ok := rc.Services[r.Service]
		if !ok {
			return nil, fmt.Errorf("route %s mengacu ke service '%s' yang belum terdaftar di 'services'", r.Prefix, r.Service)
		}
		if svc.Timeout == 0 {
			svc.Timeout = 10 * time.Second
			rc.Services[r.Service] = svc
		}
		if r.Rewrite == "" {
			rc.Routes[i].Rewrite = r.Prefix
		}
		if r.MaxBody == "" && svc.MaxBody != "" {
			rc.Routes[i].MaxBody = svc.MaxBody
		}
	}

	// Sort routes: prefix terpanjang lebih dulu agar pencocokan spesifik menang
	sort.Slice(rc.Routes, func(i, j int) bool {
		return len(rc.Routes[i].Prefix) > len(rc.Routes[j].Prefix)
	})

	return &rc, nil
}
