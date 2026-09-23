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
	NotificationServiceURL string
	FinanceServiceURL      string
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
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8083"),
		FinanceServiceURL:      getEnv("FINANCE_SERVICE_URL", "http://localhost:8085"),
		OutboxIntervalSeconds:  getEnvAsInt("OUTBOX_INTERVAL_SECONDS", 5),
		OutboxBatchSize:        getEnvAsInt("OUTBOX_BATCH_SIZE", 20),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta",
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
