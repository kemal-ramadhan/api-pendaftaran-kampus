package migrations

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

//go:embed sql/000001_create_auth_tables.up.sql
var initSchemaSQL string

// RunMigrations menjalankan SQL migration skema database
func RunMigrations(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	log.Info("Menjalankan migrasi database skema auth...")

	// Buat tabel schema_migrations jika belum ada untuk melacak migrasi
	createTrackingTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(50) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if err := db.WithContext(ctx).Exec(createTrackingTable).Error; err != nil {
		return fmt.Errorf("gagal membuat tabel tracking migrasi: %w", err)
	}

	version := "000001_create_auth_tables"
	var count int64
	err := db.WithContext(ctx).Table("schema_migrations").Where("version = ?", version).Count(&count).Error
	if err != nil {
		return fmt.Errorf("gagal memeriksa status migrasi: %w", err)
	}

	if count > 0 {
		log.Info("Migrasi sudah diterapkan sebelumnya, dilewati", zap.String("version", version))
		return nil
	}

	// Eksekusi skrip migrasi dalam transaksi
	tx := db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Eksekusi query
	statements := strings.Split(initSchemaSQL, ";")
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if err := tx.Exec(trimmed).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("gagal mengeksekusi query migrasi: %w", err)
		}
	}

	if err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal mencatat versi migrasi: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("gagal commit transaksi migrasi: %w", err)
	}

	log.Info("Berhasil menerapkan migrasi database", zap.String("version", version))
	return nil
}
