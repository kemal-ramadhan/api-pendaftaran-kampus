package seeder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var defaultRoles = []model.Role{
	{Code: model.RoleCalonMahasiswa, Name: "Calon Mahasiswa", Description: "Pendaftar mandiri calon mahasiswa baru"},
	{Code: model.RoleMahasiswa, Name: "Mahasiswa", Description: "Mahasiswa aktif terdaftar"},
	{Code: model.RoleAdminPMB, Name: "Admin PMB", Description: "Administrator seleksi dan verifikasi berkas PMB"},
	{Code: model.RoleKeuangan, Name: "Keuangan", Description: "Petugas verifikasi tagihan dan pembayaran"},
	{Code: model.RoleDosen, Name: "Dosen", Description: "Tenaga pendidik / pewawancara"},
	{Code: model.RoleProdi, Name: "Prodi", Description: "Ketua atau staf program studi"},
	{Code: model.RolePimpinan, Name: "Pimpinan Universitas", Description: "Pimpinan rektorat/fakultas"},
	{Code: model.RoleSuperadmin, Name: "Superadmin", Description: "Administrator sistem tertinggi"},
}

// SeedRoles menanam 8 role awal secara idempoten
func SeedRoles(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	for _, r := range defaultRoles {
		var existing model.Role
		err := db.WithContext(ctx).Where("code = ?", r.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			newRole := model.Role{
				Code:        r.Code,
				Name:        r.Name,
				Description: r.Description,
				CreatedAt:   time.Now(),
			}
			if err := db.WithContext(ctx).Create(&newRole).Error; err != nil {
				return fmt.Errorf("gagal seed role %s: %w", r.Code, err)
			}
			log.Info("Seeded role", zap.String("code", r.Code))
		} else if err != nil {
			return err
		}
	}
	return nil
}

// SeedSuperadmin membuat akun superadmin awal secara idempoten jika belum ada
func SeedSuperadmin(ctx context.Context, db *gorm.DB, cfg *config.Config, log *zap.Logger) error {
	email := strings.ToLower(strings.TrimSpace(cfg.SeedSuperadminEmail))
	if email == "" || cfg.SeedSuperadminPass == "" {
		log.Warn("SEED_SUPERADMIN_EMAIL atau SEED_SUPERADMIN_PASSWORD kosong, melewati seed superadmin")
		return nil
	}

	var existingUser model.User
	err := db.WithContext(ctx).Where("email = ?", email).First(&existingUser).Error
	if err == nil {
		log.Info("Akun superadmin bawaan sudah ada di database, melewati seed", zap.String("email", email))
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("gagal memeriksa keberadaan superadmin: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedSuperadminPass), cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("gagal hash password superadmin: %w", err)
	}

	now := time.Now()
	superadmin := model.User{
		ID:              uuid.New(),
		Email:           email,
		FullName:        "Super Administrator",
		PasswordHash:    string(hashedPassword),
		Status:          model.UserStatusActive,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := db.WithContext(ctx).Create(&superadmin).Error; err != nil {
		return fmt.Errorf("gagal membuat user superadmin: %w", err)
	}

	var roleSuperadmin model.Role
	if err := db.WithContext(ctx).Where("code = ?", model.RoleSuperadmin).First(&roleSuperadmin).Error; err != nil {
		return fmt.Errorf("role superadmin belum ada: %w", err)
	}

	emptyScope := ""
	userRole := model.UserRole{
		UserID:    superadmin.ID,
		RoleID:    roleSuperadmin.ID,
		ScopeType: &emptyScope,
		ScopeID:   &emptyScope,
		CreatedAt: now,
	}

	if err := db.WithContext(ctx).Create(&userRole).Error; err != nil {
		return fmt.Errorf("gagal menetapkan role superadmin: %w", err)
	}

	log.Info("Berhasil membuat akun superadmin bawaan", zap.String("email", email))
	return nil
}
