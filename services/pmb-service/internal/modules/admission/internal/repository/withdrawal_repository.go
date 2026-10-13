package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/admission/internal/model"
	"gorm.io/gorm"
)

var (
	ErrWithdrawalNotFound = errors.New("data pengunduran diri tidak ditemukan")
)

type WithdrawalRepository interface {
	Create(ctx context.Context, w *model.Withdrawal) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Withdrawal, error)
	FindLatestByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error)
	FindPendingByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error)
	FindByVerifyCodeHash(ctx context.Context, hash string) (*model.Withdrawal, error)
	FindAll(ctx context.Context, status string, page, limit int) ([]model.Withdrawal, int64, error)
	Update(ctx context.Context, w *model.Withdrawal) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
	GetNextLetterSequence(ctx context.Context, year int) (int, error)
}

type withdrawalRepository struct {
	db *gorm.DB
}

func NewWithdrawalRepository(db *gorm.DB) WithdrawalRepository {
	return &withdrawalRepository{db: db}
}

func (r *withdrawalRepository) Create(ctx context.Context, w *model.Withdrawal) error {
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *withdrawalRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Withdrawal, error) {
	var item model.Withdrawal
	err := r.db.WithContext(ctx).
		Preload("Application").
		First(&item, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *withdrawalRepository) FindLatestByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error) {
	var item model.Withdrawal
	err := r.db.WithContext(ctx).
		Preload("Application").
		Where("application_id = ?", appID).
		Order("requested_at DESC").
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *withdrawalRepository) FindPendingByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Withdrawal, error) {
	var item model.Withdrawal
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND status = ?", appID, model.WithdrawalStatusPending).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *withdrawalRepository) FindByVerifyCodeHash(ctx context.Context, hash string) (*model.Withdrawal, error) {
	var item model.Withdrawal
	err := r.db.WithContext(ctx).
		Preload("Application").
		Where("verify_code_hash = ?", hash).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWithdrawalNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *withdrawalRepository) FindAll(ctx context.Context, status string, page, limit int) ([]model.Withdrawal, int64, error) {
	var items []model.Withdrawal
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Withdrawal{})
	if status != "" {
		q = q.Where("status = ?", status)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Preload("Application").
		Order("requested_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *withdrawalRepository) Update(ctx context.Context, w *model.Withdrawal) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *withdrawalRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Hitung per status
	type StatusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusCounts []StatusCount
	_ = r.db.WithContext(ctx).Model(&model.Withdrawal{}).Select("status, count(*) as count").Group("status").Scan(&statusCounts).Error
	stats["by_status"] = statusCounts

	// Hitung per kategori alasan
	type CategoryCount struct {
		ReasonCategory string `json:"reason_category"`
		Count          int64  `json:"count"`
	}
	var categoryCounts []CategoryCount
	_ = r.db.WithContext(ctx).Model(&model.Withdrawal{}).Select("reason_category, count(*) as count").Group("reason_category").Scan(&categoryCounts).Error
	stats["by_category"] = categoryCounts

	// Hitung per tahap saat mundur
	type StageCount struct {
		StageAtRequest string `json:"stage_at_request"`
		Count          int64  `json:"count"`
	}
	var stageCounts []StageCount
	_ = r.db.WithContext(ctx).Model(&model.Withdrawal{}).Select("stage_at_request, count(*) as count").Group("stage_at_request").Scan(&stageCounts).Error
	stats["by_stage"] = stageCounts

	return stats, nil
}

func (r *withdrawalRepository) GetNextLetterSequence(ctx context.Context, year int) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Withdrawal{}).
		Where("letter_number IS NOT NULL AND EXTRACT(YEAR FROM requested_at) = ?", year).
		Count(&count).Error
	return int(count) + 1, err
}
