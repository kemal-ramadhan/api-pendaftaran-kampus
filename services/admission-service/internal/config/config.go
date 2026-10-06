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
	OrganizationServiceURL string
	NotificationServiceURL string
	FinanceServiceURL      string
	IdentityServiceURL     string
	ExamServiceURL         string
	AuthServiceURL         string
	SelectionServiceURL    string
	StudentServiceURL      string
	NIMServiceURL          string
	MinIOEndpoint          string
	MinIOAccessKey         string
	MinIOSecretKey         string
	MinIOUseSSL            bool
	DocumentBucket         string
	OutboxIntervalSeconds  int
	OutboxBatchSize        int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppPort:                getEnvAsInt("APP_PORT", 8084),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnvAsInt("DB_PORT", 5432),
		DBUser:                 getEnv("DB_USER", "pmb_user"),
		DBPassword:             getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                 getEnv("DB_NAME", "pmb_admission"),
		GatewaySharedSecret:    getEnv("GATEWAY_SHARED_SECRET", "secret-gateway-hmac"),
		InternalToken:          getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8081"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		FinanceServiceURL:      getEnv("FINANCE_SERVICE_URL", "http://localhost:8085"),
		IdentityServiceURL:     getEnv("IDENTITY_SERVICE_URL", "http://localhost:8082"),
		ExamServiceURL:         getEnv("EXAM_SERVICE_URL", "http://localhost:8087"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8080"),
		SelectionServiceURL:    getEnv("SELECTION_SERVICE_URL", "http://localhost:8090"),
		StudentServiceURL:      getEnv("STUDENT_SERVICE_URL", "http://localhost:8091"),
		NIMServiceURL:          getEnv("NIM_SERVICE_URL", "http://localhost:8092"),
		MinIOEndpoint:          getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:         getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:         getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:            getEnv("MINIO_USE_SSL", "false") == "true",
		DocumentBucket:         getEnv("DOCUMENT_BUCKET", "pmb-admission-documents"),
		OutboxIntervalSeconds:  getEnvAsInt("OUTBOX_INTERVAL_SECONDS", 1),
		OutboxBatchSize:        getEnvAsInt("OUTBOX_BATCH_SIZE", 20),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta client_encoding=UTF8",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
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
