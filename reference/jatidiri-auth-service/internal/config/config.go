package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iamhexagoninc/go-jatidiri-app/internal/model"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var (
	DB             *gorm.DB
	RDB            *redis.Client
	ctx            = context.Background()
	Logger         *zap.Logger
	Prometheus     *ginprom.Prometheus
	TracerProvider *sdktrace.TracerProvider
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system environment variables")
	}
}

func ConnectDB() {
	LoadEnv()

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	DB = db

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	db.AutoMigrate(&model.User{}, &model.Customer{})
	log.Println("Connected to PostgreSQL and migrated models")
}

func ConnectRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
	})
	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")
}

func SaveRefreshToken(userID uint, token string, duration time.Duration) error {
	return RDB.Set(ctx, getKey(userID), token, duration).Err()
}

func DeleteRefreshToken(userID uint) {
	RDB.Del(ctx, getKey(userID))
}

func ValidateRefreshToken(userID uint, token string) bool {
	val, err := RDB.Get(ctx, getKey(userID)).Result()
	if err != nil {
		return false
	}
	return val == token
}

func getKey(userID uint) string {
	return "refresh_token:user:" + fmt.Sprint(userID)
}

func RedisDB() *redis.Client { return RDB }
func Ctx() context.Context   { return ctx }

func InitLogger() {
	var err error
	Logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func InitPrometheus(r *gin.Engine) {
	Prometheus = ginprom.New(
		ginprom.Engine(r),
		ginprom.Subsystem("auth_service"),
		ginprom.Path("/metrics"),
	)
	r.Use(Prometheus.Instrument())
}

func InitTracer() {
	jaegerURL := os.Getenv("JAEGER_ENDPOINT")
	if jaegerURL == "" {
		jaegerURL = "http://localhost:14268/api/traces"
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerURL)))
	if err != nil {
		log.Fatalf("failed to create Jaeger exporter: %v", err)
	}

	TracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("jatidiri-auth-service"),
		)),
	)

	otel.SetTracerProvider(TracerProvider)
	log.Println("✅ OpenTelemetry tracing initialized")
}

func ShutdownTracer(ctx context.Context) {
	if TracerProvider != nil {
		if err := TracerProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}
