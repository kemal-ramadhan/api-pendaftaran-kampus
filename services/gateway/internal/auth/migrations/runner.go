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

//go:embed sql/000002_create_rbac_permissions.up.sql
var rbacSchemaSQL string

//go:embed sql/000003_create_sso_service_accounts.up.sql
var ssoSchemaSQL string

type migrationItem struct {
	version string
	sql     string
}

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

	migrations := []migrationItem{
		{version: "000001_create_auth_tables", sql: initSchemaSQL},
		{version: "000002_create_rbac_permissions", sql: rbacSchemaSQL},
		{version: "000003_create_sso_service_accounts", sql: ssoSchemaSQL},
	}

	for _, m := range migrations {
		var count int64
		err := db.WithContext(ctx).Table("schema_migrations").Where("version = ?", m.version).Count(&count).Error
		if err != nil {
			return fmt.Errorf("gagal memeriksa status migrasi %s: %w", m.version, err)
		}

		if count > 0 {
			log.Info("Migrasi sudah diterapkan sebelumnya, dilewati", zap.String("version", m.version))
			continue
		}

		tx := db.WithContext(ctx).Begin()
		statements := strings.Split(m.sql, ";")
		for _, stmt := range statements {
			trimmed := strings.TrimSpace(stmt)
			if trimmed == "" {
				continue
			}
			if err := tx.Exec(trimmed).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal mengeksekusi query migrasi %s: %w", m.version, err)
			}
		}

		if err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("gagal mencatat versi migrasi %s: %w", m.version, err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("gagal commit transaksi migrasi %s: %w", m.version, err)
		}

		log.Info("Berhasil menerapkan migrasi database", zap.String("version", m.version))
	}

	return nil
}
