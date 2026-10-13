package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName             string
	AppEnv              string
	AppPort             string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	GatewaySharedSecret string
	InternalToken       string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:             getEnv("APP_NAME", "organization-service"),
		AppEnv:              getEnv("APP_ENV", "development"),
		AppPort:             getEnv("APP_PORT", "8081"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "pmb_user"),
		DBPassword:          getEnv("DB_PASSWORD", "pmb_password"),
		DBName:              getEnv("DB_NAME", "pmb_organization"),
		GatewaySharedSecret: getEnv("GATEWAY_SHARED_SECRET", "secret-gateway-hmac"),
		InternalToken:       getEnv("INTERNAL_TOKEN", "secret-internal-token"),
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Jakarta client_encoding=UTF8",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
