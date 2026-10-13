package repository

import (
	"context"
	"errors"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/organization/internal/model"
	"gorm.io/gorm"
)

type InstitutionProfileRepository interface {
	Get(ctx context.Context) (*model.InstitutionProfile, error)
	Update(ctx context.Context, profile *model.InstitutionProfile) error
}

type institutionProfileRepository struct {
	db *gorm.DB
}

func NewInstitutionProfileRepository(db *gorm.DB) InstitutionProfileRepository {
	return &institutionProfileRepository{db: db}
}

func (r *institutionProfileRepository) Get(ctx context.Context) (*model.InstitutionProfile, error) {
	var profile model.InstitutionProfile
	if err := r.db.WithContext(ctx).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *institutionProfileRepository) Update(ctx context.Context, profile *model.InstitutionProfile) error {
	var existing model.InstitutionProfile
	err := r.db.WithContext(ctx).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(profile).Error
	} else if err != nil {
		return err
	}
	profile.ID = existing.ID
	return r.db.WithContext(ctx).Save(profile).Error
}
