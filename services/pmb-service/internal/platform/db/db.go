package db

import (
	"fmt"
	"time"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var AllSchemas = []string{
	"organization",
	"identity",
	"notification",
	"admission",
	"finance",
	"exam",
	"scoring",
	"selection",
	"student",
	"nim",
}

func ConnectDB(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	var gormDB *gorm.DB
	var err error

	gormCfg := &gorm.Config{
		PrepareStmt: true,
	}
	if cfg.AppEnv == "production" {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Warn)
	} else {
		gormCfg.Logger = gormlogger.Default.LogMode(gormlogger.Info)
	}

	for attempt := 1; attempt <= 30; attempt++ {
		gormDB, err = gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
		if err == nil {
			sqlDB, errPing := gormDB.DB()
			if errPing == nil && sqlDB.Ping() == nil {
				sqlDB.SetMaxIdleConns(10)
				sqlDB.SetMaxOpenConns(100)
				sqlDB.SetConnMaxLifetime(time.Hour)
				break
			}
		}
		log.Warn("Menunggu database pmb_app...", zap.Int("attempt", attempt), zap.Error(err))
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("koneksi DB pmb_app gagal: %w", err)
	}

	// Buat schema untuk setiap modul jika belum ada
	for _, s := range AllSchemas {
		if execErr := gormDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", s)).Error; execErr != nil {
			log.Warn("Gagal membuat schema modul", zap.String("schema", s), zap.Error(execErr))
		}
	}

	return gormDB, nil
}

// GetModuleDB mengembalikan instance GORM yang terisolasi untuk schema modul tertentu
func GetModuleDB(baseDB *gorm.DB, moduleSchema string) *gorm.DB {
	// Pastikan schema sudah ada
	baseDB.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", moduleSchema))

	// Clone instance DB dengan Config baru yang memiliki NamingStrategy TablePrefix = moduleSchema + "."
	cfg := *baseDB.Config
	cfg.NamingStrategy = schema.NamingStrategy{
		TablePrefix: moduleSchema + ".",
	}
	clone := baseDB.Session(&gorm.Session{NewDB: true})
	clone.Config = &cfg

	return clone
}
