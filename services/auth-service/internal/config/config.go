package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	AppName              string
	AppEnv               string
	AppPort              string
	FrontendURL          string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	RedisAddr            string
	RedisPassword        string
	JWTPrivateKeyPath    string
	JWTKid               string
	JWTIssuer            string
	JWTAudience          string
	JWTAccessTTL         time.Duration
	JWTRefreshTTL        time.Duration
	BcryptCost           int
	OTPTTL               time.Duration
	OTPSecret            string
	InternalToken        string
	GatewaySharedSecret  string
	MailDriver           string
	MailHost             string
	MailPort             int
	MailUsername         string
	MailPassword         string
	MailFromAddress      string
	MailFromName         string
	SeedSuperadminEmail  string
	SeedSuperadminPass   string
	GoogleLoginEnabled   bool
	RateLimitEnabled       bool
	OTelExporterEndpoint   string
	OrganizationServiceURL string
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}

func LoadConfig() *Config {
	return &Config{
		AppName:              getEnv("APP_NAME", "PMB Universitas"),
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:5173"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "pmb_user"),
		DBPassword:           getEnv("DB_PASSWORD", "pmb_password"),
		DBName:               getEnv("DB_NAME", "pmb_auth"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		JWTPrivateKeyPath:    getEnv("JWT_PRIVATE_KEY_PATH", ""),
		JWTKid:               getEnv("JWT_KID", "pmb-auth-1"),
		JWTIssuer:            getEnv("JWT_ISSUER", "pmb-auth-service"),
		JWTAudience:          getEnv("JWT_AUDIENCE", "pmb-api"),
		JWTAccessTTL:         getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:        getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		BcryptCost:           getIntEnv("BCRYPT_COST", 12),
		OTPTTL:               getDurationEnv("OTP_TTL", 10*time.Minute),
		OTPSecret:            getEnv("OTP_SECRET", "default-dev-otp-secret-32-bytes"),
		InternalToken:        getEnv("INTERNAL_TOKEN", "default-dev-internal-token"),
		GatewaySharedSecret:  getEnv("GATEWAY_SHARED_SECRET", "default-dev-gateway-secret"),
		MailDriver:           getEnv("MAIL_DRIVER", "log"),
		MailHost:             getEnv("MAIL_HOST", "localhost"),
		MailPort:             getIntEnv("MAIL_PORT", 1025),
		MailUsername:         getEnv("MAIL_USERNAME", ""),
		MailPassword:         getEnv("MAIL_PASSWORD", ""),
		MailFromAddress:      getEnv("MAIL_FROM_ADDRESS", "pmb@kampus.ac.id"),
		MailFromName:         getEnv("MAIL_FROM_NAME", "PMB Universitas"),
		SeedSuperadminEmail:  getEnv("SEED_SUPERADMIN_EMAIL", "superadmin@kampus.ac.id"),
		SeedSuperadminPass:   getEnv("SEED_SUPERADMIN_PASSWORD", "Superadmin123!"),
		GoogleLoginEnabled:   getBoolEnv("GOOGLE_LOGIN_ENABLED", false),
		RateLimitEnabled:       getBoolEnv("RATE_LIMIT_ENABLED", true),
		OTelExporterEndpoint:   getEnv("OTEL_EXPORTER_ENDPOINT", ""),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8081"),
	}
}

func ConnectDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
	)

	gormLogLevel := gormlogger.Silent
	if cfg.AppEnv == "development" {
		if strings.EqualFold(getEnv("LOG_LEVEL", "info"), "debug") {
			gormLogLevel = gormlogger.Info
		} else {
			gormLogLevel = gormlogger.Warn
		}
	}

	customGormLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: customGormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil instance sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func ConnectRedis(cfg *Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("gagal koneksi ke Redis di %s: %w", cfg.RedisAddr, err)
	}

	return rdb, nil
}
