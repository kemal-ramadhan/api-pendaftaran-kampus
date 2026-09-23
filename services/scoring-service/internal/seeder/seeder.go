package seeder

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"gorm.io/gorm"
)

func SeedScoreComponents(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.ScoreComponent{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultPathID := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	now := time.Now()

	components := []model.ScoreComponent{
		{
			ID:            uuid.MustParse("d0000000-0000-0000-0000-000000000001"),
			PathID:        defaultPathID,
			Code:          "written_test",
			Name:          "Ujian Tulis Berbasis Komputer",
			WeightPercent: 60.00,
			MaxScore:      100.00,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            uuid.MustParse("d0000000-0000-0000-0000-000000000002"),
			PathID:        defaultPathID,
			Code:          "interview",
			Name:          "Wawancara Motivasi & Kepribadian",
			WeightPercent: 40.00,
			MaxScore:      100.00,
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	return db.WithContext(ctx).Create(&components).Error
}
