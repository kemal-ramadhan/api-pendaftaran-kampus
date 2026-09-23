package config

import (
	"os"

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
	ExamServiceURL         string
	AdmissionServiceURL    string
	IdentityServiceURL     string
	OrganizationServiceURL string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		Port:                   getEnv("PORT", "8089"),
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "pmb_user"),
		DBPassword:             getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                 getEnv("DB_NAME", "pmb_scoring"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		InternalToken:          getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		GatewaySharedKey:       getEnv("GATEWAY_SHARED_KEY", "gateway-secret-key-minimum-32-bytes-long!"),
		ExamServiceURL:         getEnv("EXAM_SERVICE_URL", "http://localhost:8087"),
		AdmissionServiceURL:    getEnv("ADMISSION_SERVICE_URL", "http://localhost:8084"),
		IdentityServiceURL:     getEnv("IDENTITY_SERVICE_URL", "http://localhost:8082"),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8081"),
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
