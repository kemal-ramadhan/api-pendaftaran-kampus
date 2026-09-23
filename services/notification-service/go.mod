module github.com/pkm-asia/pmb-backend/services/notification-service

go 1.25.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/pkm-asia/pmb-backend/pkg v0.0.0
	go.uber.org/zap v1.27.0
	gorm.io/driver/postgres v1.6.3
	gorm.io/gorm v1.31.2
)

replace github.com/pkm-asia/pmb-backend/pkg => ../../pkg
