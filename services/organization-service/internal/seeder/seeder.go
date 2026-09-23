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

	// 1. Seed Faculties
	faculties := []model.Faculty{
		{
			Code:     "FT",
			Name:     "Fakultas Teknik",
			IsActive: true,
		},
		{
			Code:     "FEB",
			Name:     "Fakultas Ekonomi dan Bisnis",
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

	// 2. Seed Study Programs
	programs := []model.StudyProgram{
		{
			FacultyID:     facultyMap["FT"],
			Code:          "TI",
			NIMCode:       "55",
			Name:          "Teknik Informatika",
			Degree:        "S1",
			Accreditation: "Unggul",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FT"],
			Code:          "SI",
			NIMCode:       "57",
			Name:          "Sistem Informasi",
			Degree:        "S1",
			Accreditation: "Baik Sekali",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FEB"],
			Code:          "MN",
			NIMCode:       "61",
			Name:          "Manajemen",
			Degree:        "S1",
			Accreditation: "A",
			IsActive:      true,
		},
		{
			FacultyID:     facultyMap["FEB"],
			Code:          "AK",
			NIMCode:       "62",
			Name:          "Akuntansi",
			Degree:        "S1",
			Accreditation: "A",
			IsActive:      true,
		},
	}

	for _, p := range programs {
		var existing model.StudyProgram
		err := db.WithContext(ctx).Where("code = ?", p.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			p.CreatedAt = now
			p.UpdatedAt = now
			if err := db.WithContext(ctx).Create(&p).Error; err != nil {
				return err
			}
			logger.Info("Program Studi berhasil di-seed", zap.String("code", p.Code), zap.String("name", p.Name))
		} else if err != nil {
			return err
		}
	}

	logger.Info("Seeding Organization Service selesai")
	return nil
}
