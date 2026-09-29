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
	{Code: model.RoleDekan, Name: "Dekan", Description: "Pimpinan fakultas / dekanat"},
	{Code: model.RolePimpinan, Name: "Pimpinan Universitas", Description: "Pimpinan rektorat/fakultas"},
	{Code: model.RoleSuperadmin, Name: "Superadmin", Description: "Administrator sistem tertinggi"},
}

// SeedRoles menanam role awal secara idempoten
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

var defaultPermissions = []model.Permission{
	{Code: "dashboard.overview.read", Module: "dashboard", Description: "Melihat ringkasan eksekutif dan KPI operasional PMB"},
	{Code: "dashboard.institution.read", Module: "dashboard", Description: "Melihat dashboard institusional dan analitik fakultas/prodi"},
	{Code: "admission.document.verify", Module: "admission", Description: "Memvalidasi, menerima, atau meminta revisi berkas pendaftar"},
	{Code: "admission.path_change.manage", Module: "admission", Description: "Meninjau dan menyetujui pengajuan pindah jalur pendaftaran"},
	{Code: "admission.withdrawal.manage", Module: "admission", Description: "Meninjau dan memproses permohonan pengunduran diri"},
	{Code: "finance.reconciliation.read", Module: "finance", Description: "Melihat data rekonsiliasi dan transaksi pembayaran VA"},
	{Code: "finance.payment.manual_confirm", Module: "finance", Description: "Melakukan konfirmasi pelunasan tagihan secara manual"},
	{Code: "finance.ledger.read", Module: "finance", Description: "Melihat buku besar arus kas masuk PMB"},
	{Code: "finance.fee.manage", Module: "finance", Description: "Mengelola skema tarif formulir dan UKT program studi"},
	{Code: "selection.score.read", Module: "selection", Description: "Melihat skor ujian CBT, rapor, dan scoreboard seleksi"},
	{Code: "selection.interview.score", Module: "selection", Description: "Menginput dan memperbarui nilai wawancara calon mahasiswa"},
	{Code: "selection.recommendation.manage", Module: "selection", Description: "Mengelola rekomendasi kelulusan prodi"},
	{Code: "selection.approval.decide", Module: "selection", Description: "Memberikan persetujuan akhir kelulusan seleksi"},
	{Code: "selection.announcement.publish", Module: "selection", Description: "Mempublikasikan pengumuman kelulusan resmi"},
	{Code: "student.nim.issue", Module: "student", Description: "Menetapkan dan menerbitkan Nomor Induk Mahasiswa (NIM)"},
	{Code: "student.pkkmb.manage", Module: "student", Description: "Mengelola kelompok dan penugasan gugus PKKMB"},
	{Code: "student.activation.manage", Module: "student", Description: "Mengelola ekspor data aktivasi SIAKAD"},
	{Code: "report.export", Module: "report", Description: "Mengekspor laporan pendaftar, seleksi, dan keuangan"},
	{Code: "report.official.print", Module: "report", Description: "Mencetak dokumen resmi Berita Acara PMB dalam format PDF"},
	{Code: "broadcast.send", Module: "notification", Description: "Membuat template dan mengirim broadcast notifikasi massal"},
	{Code: "organization.program.read", Module: "organization", Description: "Melihat master data fakultas dan program studi"},
	{Code: "organization.program.manage", Module: "organization", Description: "Mengelola pembukaan pendaftaran dan data prodi"},
	{Code: "auth.user.read", Module: "auth", Description: "Melihat daftar pengguna dan staf"},
	{Code: "auth.user.manage", Module: "auth", Description: "Mengelola akun pengguna dan penetapan role"},
	{Code: "auth.permission.read", Module: "auth", Description: "Melihat daftar permission dan pemetaan role"},
	{Code: "auth.permission.manage", Module: "auth", Description: "Mengubah pemetaan permission role"},
	{Code: "auth.role.manage", Module: "auth", Description: "Mengelola master role sistem"},
}

var rolePermissionMap = map[string][]string{
	model.RoleSuperadmin: {
		"dashboard.overview.read", "dashboard.institution.read",
		"admission.document.verify", "admission.path_change.manage", "admission.withdrawal.manage",
		"finance.reconciliation.read", "finance.payment.manual_confirm", "finance.ledger.read", "finance.fee.manage",
		"selection.score.read", "selection.interview.score", "selection.recommendation.manage", "selection.approval.decide", "selection.announcement.publish",
		"student.nim.issue", "student.pkkmb.manage", "student.activation.manage",
		"report.export", "report.official.print",
		"broadcast.send",
		"organization.program.read", "organization.program.manage",
		"auth.user.read", "auth.user.manage", "auth.permission.read", "auth.permission.manage", "auth.role.manage",
	},
	model.RoleAdminPMB: {
		"dashboard.overview.read", "dashboard.institution.read",
		"admission.document.verify", "admission.path_change.manage", "admission.withdrawal.manage",
		"finance.reconciliation.read", "finance.ledger.read", "finance.fee.manage",
		"selection.score.read", "selection.interview.score", "selection.recommendation.manage", "selection.announcement.publish",
		"student.nim.issue", "student.pkkmb.manage", "student.activation.manage",
		"report.export", "report.official.print",
		"broadcast.send",
		"organization.program.read",
		"auth.user.read",
	},
	model.RoleKeuangan: {
		"dashboard.overview.read", "dashboard.institution.read",
		"finance.reconciliation.read", "finance.payment.manual_confirm", "finance.ledger.read", "finance.fee.manage",
		"student.nim.issue",
		"report.export",
		"organization.program.read",
	},
	model.RoleProdi: {
		"dashboard.overview.read", "dashboard.institution.read",
		"admission.document.verify",
		"selection.score.read", "selection.recommendation.manage",
		"student.nim.issue",
		"report.export",
		"organization.program.read",
	},
	model.RoleDekan: {
		"dashboard.overview.read", "dashboard.institution.read",
		"selection.score.read",
		"report.export",
		"organization.program.read",
	},
	model.RolePimpinan: {
		"dashboard.overview.read", "dashboard.institution.read",
		"finance.reconciliation.read",
		"selection.score.read", "selection.approval.decide",
		"student.nim.issue",
		"report.export",
		"organization.program.read",
		"auth.user.read",
	},
	model.RoleDosen: {
		"selection.score.read",
		"selection.interview.score",
	},
}

// SeedPermissions menanam daftar permission dan relasi role_permissions awal sesuai matriks 1.4
func SeedPermissions(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	for _, p := range defaultPermissions {
		var existing model.Permission
		err := db.WithContext(ctx).Where("code = ?", p.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			newPerm := model.Permission{
				Code:        p.Code,
				Module:      p.Module,
				Description: p.Description,
				CreatedAt:   time.Now(),
			}
			if err := db.WithContext(ctx).Create(&newPerm).Error; err != nil {
				return fmt.Errorf("gagal seed permission %s: %w", p.Code, err)
			}
		} else if err != nil {
			return err
		}
	}

	for roleCode, permCodes := range rolePermissionMap {
		var r model.Role
		if err := db.WithContext(ctx).Where("code = ?", roleCode).First(&r).Error; err != nil {
			log.Warn("Role belum ada saat mapping permission, dilewati", zap.String("role", roleCode))
			continue
		}

		for _, pc := range permCodes {
			var rp model.RolePermission
			err := db.WithContext(ctx).Where("role_id = ? AND permission_code = ?", r.ID, pc).First(&rp).Error
			if err == gorm.ErrRecordNotFound {
				newRP := model.RolePermission{
					RoleID:         r.ID,
					PermissionCode: pc,
					CreatedAt:      time.Now(),
				}
				if err := db.WithContext(ctx).Create(&newRP).Error; err != nil {
					return fmt.Errorf("gagal seed role_permission (%s, %s): %w", roleCode, pc, err)
				}
			} else if err != nil {
				return err
			}
		}
	}

	log.Info("Berhasil menanam permission & pemetaan role_permissions RBAC")
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
