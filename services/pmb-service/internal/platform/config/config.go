package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppName string
	AppEnv  string
	AppPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisAddr     string
	RedisPassword string

	// MinIO
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool

	// Security
	GatewaySharedSecret string
	InternalToken       string

	// Downstream/External services
	GatewayInternalURL string
	OCRServiceURL      string
	OCREnabled         bool

	// Outbox & Worker
	OutboxIntervalSeconds int
	OutboxBatchSize       int

	// SMTP / Mailpit
	SMTPHost      string
	SMTPPort      int
	SMTPFromEmail string
	SMTPFromName  string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		AppName:             getEnv("APP_NAME", "pmb-service"),
		AppEnv:              getEnv("APP_ENV", "development"),
		AppPort:             getEnv("PORT", getEnv("APP_PORT", "8080")),
		DBHost:              getEnv("DB_HOST", "postgres"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "pmb_user"),
		DBPassword:          getEnv("DB_PASSWORD", "pmb_password"),
		DBName:              getEnv("DB_NAME", "pmb_app"),
		DBSSLMode:           getEnv("DB_SSLMODE", "disable"),
		RedisAddr:           getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		MinioEndpoint:       getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", getEnv("MINIO_ROOT_USER", "minioadmin")),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", getEnv("MINIO_ROOT_PASSWORD", "minioadmin")),
		MinioUseSSL:         getEnv("MINIO_USE_SSL", "false") == "true",
		GatewaySharedSecret: getEnv("GATEWAY_SHARED_SECRET", getEnv("GATEWAY_SHARED_KEY", "secret-gateway-hmac")),
		InternalToken:       getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		GatewayInternalURL:  getEnv("GATEWAY_INTERNAL_URL", "http://gateway:8081"),
		OCRServiceURL:       getEnv("OCR_SERVICE_URL", "http://ocr-service:8088"),
		OCREnabled:          getEnv("OCR_ENABLED", "true") != "false",
		SMTPHost:            getEnv("SMTP_HOST", "mailpit"),
		SMTPFromEmail:       getEnv("SMTP_FROM_EMAIL", "noreply@pmb.ac.id"),
		SMTPFromName:        getEnv("SMTP_FROM_NAME", "PMB Universitas"),
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	if err != nil {
		smtpPort = 1025
	}
	cfg.SMTPPort = smtpPort

	interval, err := strconv.Atoi(getEnv("OUTBOX_INTERVAL_SECONDS", "1"))
	if err != nil {
		interval = 1
	}
	cfg.OutboxIntervalSeconds = interval

	batch, err := strconv.Atoi(getEnv("OUTBOX_BATCH_SIZE", "20"))
	if err != nil {
		batch = 20
	}
	cfg.OutboxBatchSize = batch

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}
