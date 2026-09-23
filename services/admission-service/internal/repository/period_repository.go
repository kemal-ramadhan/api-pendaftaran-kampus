package repository

import (
	"context"
	"errors"
	"time"

	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrPeriodNotFound = errors.New("periode pendaftaran tidak ditemukan")
)

type PeriodRepository interface {
	FindActive(ctx context.Context) (*model.Period, error)
	FindByID(ctx context.Context, id uint) (*model.Period, error)
	FindAll(ctx context.Context) ([]model.Period, error)
}

type periodRepository struct {
	db *gorm.DB
}

func NewPeriodRepository(db *gorm.DB) PeriodRepository {
	return &periodRepository{db: db}
}

func (r *periodRepository) FindActive(ctx context.Context) (*model.Period, error) {
	var p model.Period
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Order("id desc").
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPeriodNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *periodRepository) FindByID(ctx context.Context, id uint) (*model.Period, error) {
	var p model.Period
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPeriodNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *periodRepository) FindAll(ctx context.Context) ([]model.Period, error) {
	var list []model.Period
	err := r.db.WithContext(ctx).Order("id desc").Find(&list).Error
	return list, err
}
