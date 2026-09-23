package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AddressRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error)
	FindByUserIDAndType(ctx context.Context, userID uuid.UUID, addrType string) (*model.Address, error)
	Save(ctx context.Context, address *model.Address) error
}

type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) AddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error) {
	var addrs []model.Address
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&addrs).Error
	return addrs, err
}

func (r *addressRepository) FindByUserIDAndType(ctx context.Context, userID uuid.UUID, addrType string) (*model.Address, error) {
	var addr model.Address
	err := r.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, addrType).First(&addr).Error
	if err != nil {
		return nil, err
	}
	return &addr, nil
}

func (r *addressRepository) Save(ctx context.Context, address *model.Address) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "type"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"street", "rt", "rw", "village", "district",
			"city", "province", "postal_code", "updated_at",
		}),
	}).Save(address).Error
}
