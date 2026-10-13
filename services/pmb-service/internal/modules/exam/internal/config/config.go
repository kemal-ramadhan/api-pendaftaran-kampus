package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Port                   string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	RedisAddr              string
	InternalToken          string
	GatewaySharedKey       string
	MinIOEndpoint          string
	MinIOAccessKey         string
	MinIOSecretKey         string
	MinIOUseSSL            bool
	ExamCardBucket         string
	AdmissionServiceURL    string
	IdentityServiceURL     string
	NotificationServiceURL string
	ScoringServiceURL      string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	minioSSL, _ := strconv.ParseBool(getEnv("MINIO_USE_SSL", "false"))

	return &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		Port:                   getEnv("PORT", "8087"),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "pmb_user"),
		DBPassword:             getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                 getEnv("DB_NAME", "pmb_exam"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		InternalToken:          getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		GatewaySharedKey:       getEnv("GATEWAY_SHARED_SECRET", getEnv("GATEWAY_SHARED_KEY", "change-this-gateway-shared-secret")),
		MinIOEndpoint:          getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:         getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:         getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:            minioSSL,
		ExamCardBucket:         getEnv("EXAM_CARD_BUCKET", "pmb-exam-cards"),
		AdmissionServiceURL:    getEnv("ADMISSION_SERVICE_URL", "http://localhost:8086"),
		IdentityServiceURL:     getEnv("IDENTITY_SERVICE_URL", "http://localhost:8083"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8084"),
		ScoringServiceURL:      getEnv("SCORING_SERVICE_URL", "http://localhost:8088"),
	}
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
