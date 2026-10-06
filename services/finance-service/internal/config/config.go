package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	AppPort                int
	DBHost                 string
	DBPort                 int
	DBUser                 string
	DBPassword             string
	DBName                 string
	GatewaySharedSecret    string
	InternalToken          string
	AdmissionServiceURL    string
	NotificationServiceURL string
	PaymentWebhookSecret   string
	PaymentProvider           string // mock | duitku
	DuitkuEnv                 string // sandbox | production
	DuitkuMerchantCode        string
	DuitkuApiKey              string
	DuitkuExpiryMinutes       int
	DuitkuCallbackURL         string
	DuitkuReturnURL           string
	DuitkuHTTPTimeout         string
	DuitkuPOPBaseURL          string
	DuitkuWebAPIBaseURL       string
	DuitkuReconcileInterval   string
	DuitkuCallbackIPAllowlist string
	AllowSandboxInProduction  bool
	ReconciliationIntervalMinutes int
	MinioEndpoint             string
	MinioAccessKey            string
	MinioSecretKey            string
	MinioUseSSL               bool
	MinioPublicEndpoint       string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	duitkuEnv := getEnv("DUITKU_ENV", "sandbox")
	defaultPOP := "https://api-sandbox.duitku.com"
	defaultWebAPI := "https://sandbox.duitku.com"
	if strings.ToLower(duitkuEnv) == "production" {
		defaultPOP = "https://api-prod.duitku.com"
		defaultWebAPI = "https://passport.duitku.com"
	}

	cfg := &Config{
		AppEnv:                    getEnv("APP_ENV", "development"),
		AppPort:                   getEnvAsInt("APP_PORT", 8085),
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnvAsInt("DB_PORT", 5432),
		DBUser:                    getEnv("DB_USER", "pmb_user"),
		DBPassword:                getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                    getEnv("DB_NAME", "pmb_finance"),
		GatewaySharedSecret:       getEnv("GATEWAY_SHARED_SECRET", "secret-gateway-hmac"),
		InternalToken:             getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		AdmissionServiceURL:       getEnv("ADMISSION_SERVICE_URL", "http://localhost:8084"),
		NotificationServiceURL:    getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		PaymentWebhookSecret:      getEnv("PAYMENT_WEBHOOK_SECRET", getEnv("WEBHOOK_SECRET_MOCK", "mock-secret-key-123")),
		PaymentProvider:           getEnv("PAYMENT_PROVIDER", "mock"),
		DuitkuEnv:                 duitkuEnv,
		DuitkuMerchantCode:        getEnv("DUITKU_MERCHANT_CODE", ""),
		DuitkuApiKey:              getEnv("DUITKU_API_KEY", ""),
		DuitkuExpiryMinutes:       getEnvAsInt("DUITKU_EXPIRY_MINUTES", 1440),
		DuitkuCallbackURL:         getEnv("DUITKU_CALLBACK_URL", ""),
		DuitkuReturnURL:           getEnv("DUITKU_RETURN_URL", ""),
		DuitkuHTTPTimeout:             getEnv("DUITKU_HTTP_TIMEOUT", "15s"),
		DuitkuPOPBaseURL:              getEnv("DUITKU_POP_BASE_URL", defaultPOP),
		DuitkuWebAPIBaseURL:           getEnv("DUITKU_WEBAPI_BASE_URL", defaultWebAPI),
		DuitkuReconcileInterval:       getEnv("DUITKU_RECONCILE_INTERVAL", "10m"),
		DuitkuCallbackIPAllowlist:     getEnv("DUITKU_CALLBACK_IP_ALLOWLIST", ""),
		AllowSandboxInProduction:      getEnvAsBool("ALLOW_SANDBOX_IN_PRODUCTION", false),
		ReconciliationIntervalMinutes: getEnvAsInt("RECONCILIATION_INTERVAL_MINUTES", 10),
		MinioEndpoint:                 getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:                getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:                getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:                   getEnvAsBool("MINIO_USE_SSL", false),
		MinioPublicEndpoint:           getEnv("MINIO_PUBLIC_ENDPOINT", ""),
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.AppEnv == "production" && c.PaymentProvider == "duitku" {
		if strings.ToLower(c.DuitkuEnv) != "production" && !c.AllowSandboxInProduction {
			return fmt.Errorf("DUITKU_ENV harus 'production' pada lingkungan produksi kecuali ALLOW_SANDBOX_IN_PRODUCTION=true")
		}
		if c.DuitkuMerchantCode == "" || c.DuitkuMerchantCode == "DXXXX" {
			return fmt.Errorf("DUITKU_MERCHANT_CODE wajib diisi untuk produksi")
		}
		if c.DuitkuApiKey == "" || c.DuitkuApiKey == "secret" {
			return fmt.Errorf("DUITKU_API_KEY wajib diisi untuk produksi")
		}
	}
	return nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta client_encoding=UTF8",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
