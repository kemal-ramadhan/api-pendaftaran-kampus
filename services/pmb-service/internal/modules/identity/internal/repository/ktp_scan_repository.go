package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/model"
	"gorm.io/gorm"
)

var (
	ErrKTPScanNotFound = errors.New("data pemindaian KTP tidak ditemukan")
)

type KTPScanRepository interface {
	Create(ctx context.Context, scan *model.KTPScan) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.KTPScan, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.KTPScan, error)
	Update(ctx context.Context, scan *model.KTPScan) error
	CountScansSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
}

type ktpScanRepository struct {
	db *gorm.DB
}

func NewKTPScanRepository(db *gorm.DB) KTPScanRepository {
	return &ktpScanRepository{db: db}
}

func (r *ktpScanRepository) Create(ctx context.Context, scan *model.KTPScan) error {
	if scan.ID == uuid.Nil {
		scan.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(scan).Error
}

func (r *ktpScanRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.KTPScan, error) {
	var scan model.KTPScan
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&scan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKTPScanNotFound
		}
		return nil, err
	}
	return &scan, nil
}

func (r *ktpScanRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.KTPScan, error) {
	var scans []model.KTPScan
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&scans).Error
	return scans, err
}

func (r *ktpScanRepository) Update(ctx context.Context, scan *model.KTPScan) error {
	return r.db.WithContext(ctx).Save(scan).Error
}

func (r *ktpScanRepository) CountScansSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.KTPScan{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&count).Error
	return count, err
}
