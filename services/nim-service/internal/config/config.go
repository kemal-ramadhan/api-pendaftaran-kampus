package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv        string
	Port          string
	IsDevelopment bool

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	GatewaySecret string
	InternalToken string

	StudentServiceURL      string
	AuthServiceURL         string
	AdmissionServiceURL    string
	NotificationServiceURL string
	OrganizationServiceURL string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	appEnv := getEnv("APP_ENV", "development")
	cfg := &Config{
		AppEnv:        appEnv,
		Port:          getEnv("PORT", "8092"),
		IsDevelopment: appEnv == "development" || appEnv == "test",

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "pmb_nim"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		GatewaySecret: getEnv("GATEWAY_SHARED_SECRET", "secret-gateway-hmac"),
		InternalToken: getEnv("INTERNAL_TOKEN", "secret-internal-token"),

		StudentServiceURL:      getEnv("STUDENT_SERVICE_URL", "http://localhost:8091"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8080"),
		AdmissionServiceURL:    getEnv("ADMISSION_SERVICE_URL", "http://localhost:8084"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8081"),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
