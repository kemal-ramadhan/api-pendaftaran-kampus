package seeder

import (
	"context"
	"time"

	"github.com/pkm-asia/pmb-backend/services/organization-service/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Seed(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	logger.Info("Memeriksa seeder Organization Service...")

	// 1. Seed Faculties Universitas Bakti Tunas Husada
	faculties := []model.Faculty{
		{
			Code:     "FF",
			Name:     "Fakultas Farmasi",
			IsActive: true,
		},
		{
			Code:     "FIK",
			Name:     "Fakultas Ilmu Kesehatan",
			IsActive: true,
		},
		{
			Code:     "FTB",
			Name:     "Fakultas Teknologi & Bisnis",
			IsActive: true,
		},
	}

	facultyMap := make(map[string]uint)
	now := time.Now()

	for _, f := range faculties {
		var existing model.Faculty
		err := db.WithContext(ctx).Where("code = ?", f.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			f.CreatedAt = now
			f.UpdatedAt = now
			if err := db.WithContext(ctx).Create(&f).Error; err != nil {
				return err
			}
			facultyMap[f.Code] = f.ID
			logger.Info("Fakultas berhasil di-seed", zap.String("code", f.Code), zap.String("name", f.Name))
		} else if err == nil {
			facultyMap[f.Code] = existing.ID
		} else {
			return err
		}
	}

	// 2. Seed Study Programs Universitas BTH
	programs := []model.StudyProgram{
		{
			FacultyID:     facultyMap["FF"],
			Code:          "FARM",
			NIMCode:       "01",
			Name:          "S1 Farmasi",
			Degree:        "S1",
			Accreditation: "Unggul",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FTB"],
			Code:          "TI",
			NIMCode:       "55",
			Name:          "S1 Teknologi Informasi",
			Degree:        "S1",
			Accreditation: "Baik Sekali",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FIK"],
			Code:          "ARS",
			NIMCode:       "03",
			Name:          "S1 Administrasi Rumah Sakit",
			Degree:        "S1",
			Accreditation: "Baik Sekali",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FIK"],
			Code:          "TLM",
			NIMCode:       "04",
			Name:          "D3 Analis Kesehatan (TLM)",
			Degree:        "D3",
			Accreditation: "Baik Sekali",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FF"],
			Code:          "DFARM",
			NIMCode:       "05",
			Name:          "D3 Farmasi",
			Degree:        "D3",
			Accreditation: "Unggul",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FTB"],
			Code:          "BD",
			NIMCode:       "57",
			Name:          "S1 Bisnis Digital",
			Degree:        "S1",
			Accreditation: "Baik",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FIK"],
			Code:          "SKEP",
			NIMCode:       "09",
			Name:          "S1 Keperawatan",
			Degree:        "S1",
			Accreditation: "Baik Sekali",
			IsActive:      true,
			AdmissionOpen: true,
		},
	}

	for _, p := range programs {
		var existing model.StudyProgram
		err := db.WithContext(ctx).Where("code = ?", p.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			p.CreatedAt = now
			p.UpdatedAt = now
			p.AdmissionOpen = true
			if err := db.WithContext(ctx).Create(&p).Error; err != nil {
				return err
			}
			logger.Info("Program Studi berhasil di-seed", zap.String("code", p.Code), zap.String("name", p.Name))
		} else if err != nil {
			return err
		}
	}

	// 3. Seed Institution Profile
	var existingProfile model.InstitutionProfile
	err := db.WithContext(ctx).First(&existingProfile).Error
	if err == gorm.ErrRecordNotFound {
		defaultProfile := model.InstitutionProfile{
			FoundationName:      "Yayasan Pendidikan Bakti Nusantara",
			InstitutionName:     "Universitas Bakti Teknologi Nusantara",
			DocumentCode:        "UBTH",
			Address:             "Jl. Jenderal Sudirman No. 123",
			City:                "Bandung",
			Email:               "pmb@ubth.ac.id",
			Phone:               "(022) 1234567",
			Hotline:             "0812-3456-7890",
			ServiceHours:        "Senin–Jumat 08.00–16.00 WIB | Sabtu 08.00–13.00 WIB",
			LogoURL:             "https://desainpakeai.com/assets/logo.png",
			CommitteeChairName:  "Dr. Ir. H. Ahmad Sudrajat, M.T.",
			CommitteeChairTitle: "Ketua Panitia PMB 2026",
			CommitteeChairNIP:   "197805122003121002",
			BankAccounts: model.BankAccounts{
				{
					BankName:      "Bank Mandiri",
					AccountNumber: "1310012345678",
					AccountName:   "Yayasan Bakti Teknologi PMB",
					IsPrimary:     true,
				},
				{
					BankName:      "Bank Central Asia (BCA)",
					AccountNumber: "7820123456",
					AccountName:   "Yayasan Bakti Teknologi PMB",
					IsPrimary:     false,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := db.WithContext(ctx).Create(&defaultProfile).Error; err != nil {
			return err
		}
		logger.Info("Profil Institusi berhasil di-seed", zap.String("code", defaultProfile.DocumentCode))
	} else if err != nil {
		return err
	}

	logger.Info("Seeding Organization Service selesai")
	return nil
}
