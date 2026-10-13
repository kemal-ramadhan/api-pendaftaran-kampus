package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GuardianRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Guardian, error)
	FindByUserIDAndRelation(ctx context.Context, userID uuid.UUID, relation string) (*model.Guardian, error)
	Save(ctx context.Context, guardian *model.Guardian) error
}

type guardianRepository struct {
	db *gorm.DB
}

func NewGuardianRepository(db *gorm.DB) GuardianRepository {
	return &guardianRepository{db: db}
}

func (r *guardianRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Guardian, error) {
	var list []model.Guardian
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&list).Error
	return list, err
}

func (r *guardianRepository) FindByUserIDAndRelation(ctx context.Context, userID uuid.UUID, relation string) (*model.Guardian, error) {
	var g model.Guardian
	err := r.db.WithContext(ctx).Where("user_id = ? AND relation = ?", userID, relation).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *guardianRepository) Save(ctx context.Context, guardian *model.Guardian) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "relation"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "occupation", "income_range", "phone", "updated_at",
		}),
	}).Save(guardian).Error
}
