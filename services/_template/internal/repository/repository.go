package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/pagination"
	"github.com/pkm-asia/pmb-backend/services/_template/internal/model"
	"gorm.io/gorm"
)

type TemplateRepository interface {
	FindAll(ctx context.Context, q pagination.Query) ([]model.Item, int64, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Item, error)
	Create(ctx context.Context, item *model.Item) error
}

type templateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) FindAll(ctx context.Context, q pagination.Query) ([]model.Item, int64, error) {
	var items []model.Item
	var total int64

	tx := r.db.WithContext(ctx).Model(&model.Item{})
	if q.Search != "" {
		tx = tx.Where("name ILIKE ?", "%"+q.Search+"%")
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := tx.Scopes(pagination.Scope(q)).Find(&items).Error
	return items, total, err
}

func (r *templateRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Item, error) {
	var item model.Item
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *templateRepository) Create(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Create(item).Error
}
