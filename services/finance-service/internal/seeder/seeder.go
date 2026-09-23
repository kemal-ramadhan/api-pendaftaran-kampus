package seeder

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

func SeedFeeSchemes(ctx context.Context, db *gorm.DB) error {
	var count int64
	db.WithContext(ctx).Model(&model.FeeScheme{}).Count(&count)
	if count > 0 {
		return nil
	}

	schemes := []model.FeeScheme{
		{
			ID:                     uuid.New(),
			PeriodID:               1,
			AdmissionPathID:        1, // Reguler
			FeeType:                "registration",
			Amount:                 250000,
			InstallmentAllowed:     false,
			MaxInstallments:        1,
			MinFirstPaymentPercent: 100,
		},
		{
			ID:                     uuid.New(),
			PeriodID:               1,
			AdmissionPathID:        2, // Prestasi Akademik
			FeeType:                "registration",
			Amount:                 200000,
			InstallmentAllowed:     false,
			MaxInstallments:        1,
			MinFirstPaymentPercent: 100,
		},
		{
			ID:                     uuid.New(),
			PeriodID:               1,
			AdmissionPathID:        3, // Prestasi Non-Akademik
			FeeType:                "registration",
			Amount:                 200000,
			InstallmentAllowed:     false,
			MaxInstallments:        1,
			MinFirstPaymentPercent: 100,
		},
		{
			ID:                     uuid.New(),
			PeriodID:               1,
			AdmissionPathID:        4, // Beasiswa
			FeeType:                "registration",
			Amount:                 0,
			InstallmentAllowed:     false,
			MaxInstallments:        1,
			MinFirstPaymentPercent: 100,
		},
	}

	for _, s := range schemes {
		item := s
		if err := db.WithContext(ctx).Create(&item).Error; err != nil {
			return err
		}
	}

	log.Println("[FINANCE-SEEDER] Berhasil seed skema biaya registrasi awal")
	return nil
}
