package seeder

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"gorm.io/gorm"
)

func SeedExamSessions(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.ExamSession{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	loc := "Gedung Ujian Terpadu PMB Lt. 2 R.201"
	session := model.ExamSession{
		ID:       uuid.MustParse("a0000000-0000-0000-0000-000000000001"),
		PeriodID: uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
		PathID:   uuid.MustParse("c0000000-0000-0000-0000-000000000001"),
		Type:     model.SessionTypeWritten,
		Title:    "Ujian Tulis Kemampuan Dasar & Skolastik",
		StartAt:  time.Now().Add(48 * time.Hour),
		EndAt:    time.Now().Add(50 * time.Hour),
		Location: &loc,
		Capacity: 30,
		IsActive: true,
	}

	return db.WithContext(ctx).Create(&session).Error
}
