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

	AdmissionServiceURL    string
	SelectionServiceURL    string
	FinanceServiceURL      string
	OrganizationServiceURL string
	NIMServiceURL          string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	appEnv := getEnv("APP_ENV", "development")
	cfg := &Config{
		AppEnv:        appEnv,
		Port:          getEnv("PORT", "8091"),
		IsDevelopment: appEnv == "development" || appEnv == "test",

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "pmb_user"),
		DBPassword: getEnv("DB_PASSWORD", "pmb_password"),
		DBName:     getEnv("DB_NAME", "pmb_student"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		GatewaySecret: getEnv("GATEWAY_SHARED_KEY", "secret-gateway-hmac"),
		InternalToken: getEnv("INTERNAL_TOKEN", "secret-internal-token"),

		AdmissionServiceURL:    getEnv("ADMISSION_SERVICE_URL", "http://localhost:8084"),
		SelectionServiceURL:    getEnv("SELECTION_SERVICE_URL", "http://localhost:8090"),
		FinanceServiceURL:      getEnv("FINANCE_SERVICE_URL", "http://localhost:8085"),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8081"),
		NIMServiceURL:          getEnv("NIM_SERVICE_URL", "http://localhost:8092"),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
