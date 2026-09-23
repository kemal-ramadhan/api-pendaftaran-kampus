package config

import (
	"fmt"
	"os"
	"strconv"

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
	MinioEndpoint          string
	MinioAccessKey         string
	MinioSecretKey         string
	MinioUseSSL            bool
	MinioPublicEndpoint    string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppPort:                getEnvAsInt("APP_PORT", 8085),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnvAsInt("DB_PORT", 5432),
		DBUser:                 getEnv("DB_USER", "pmb_user"),
		DBPassword:             getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                 getEnv("DB_NAME", "pmb_finance"),
		GatewaySharedSecret:    getEnv("GATEWAY_SHARED_SECRET", "secret-gateway-hmac"),
		InternalToken:          getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		AdmissionServiceURL:    getEnv("ADMISSION_SERVICE_URL", "http://localhost:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		PaymentWebhookSecret:   getEnv("PAYMENT_WEBHOOK_SECRET", "secret-payment-webhook"),
		MinioEndpoint:          getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:         getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:         getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:            getEnvAsBool("MINIO_USE_SSL", false),
		MinioPublicEndpoint:    getEnv("MINIO_PUBLIC_ENDPOINT", ""),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta",
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
