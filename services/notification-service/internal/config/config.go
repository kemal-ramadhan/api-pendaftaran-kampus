package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	AppPort        int
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	InternalToken       string
	GatewaySharedSecret string

	// SMTP config (default to Mailpit)
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPFromName  string

	// WhatsApp config
	WADriver   string // "log" or "mock" or "http"
	WAEndpoint string
	WAToken    string

	// Worker config
	WorkerIntervalSeconds int
	WorkerBatchSize       int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                getEnv("APP_ENV", "development"),
		AppPort:               getEnvAsInt("APP_PORT", 8083),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnvAsInt("DB_PORT", 5432),
		DBUser:                getEnv("DB_USER", "pmb_user"),
		DBPassword:            getEnv("DB_PASSWORD", "pmb_password"),
		DBName:                getEnv("DB_NAME", "pmb_notification"),
		InternalToken:         getEnv("INTERNAL_TOKEN", "secret-internal-token"),
		GatewaySharedSecret:   getEnv("GATEWAY_SHARED_SECRET", "gateway-secret-key-12345"),
		SMTPHost:              getEnv("SMTP_HOST", "localhost"),
		SMTPPort:              getEnvAsInt("SMTP_PORT", 1025),
		SMTPUser:              getEnv("SMTP_USER", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:         getEnv("SMTP_FROM_EMAIL", "noreply@pmb.ac.id"),
		SMTPFromName:          getEnv("SMTP_FROM_NAME", "PMB Universitas"),
		WADriver:              getEnv("WA_DRIVER", "log"),
		WAEndpoint:            getEnv("WA_ENDPOINT", ""),
		WAToken:               getEnv("WA_TOKEN", ""),
		WorkerIntervalSeconds: getEnvAsInt("WORKER_INTERVAL_SECONDS", 2),
		WorkerBatchSize:       getEnvAsInt("WORKER_BATCH_SIZE", 20),
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
