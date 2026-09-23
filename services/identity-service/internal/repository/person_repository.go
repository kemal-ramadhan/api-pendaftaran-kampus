package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPersonNotFound = errors.New("data person tidak ditemukan")
	ErrNIKDuplicate   = errors.New("NIK sudah terdaftar pada pengguna lain")
)

type PersonRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Person, error)
	FindByNIKHash(ctx context.Context, nikHash string) (*model.Person, error)
	Save(ctx context.Context, person *model.Person) error
	UpdatePhoto(ctx context.Context, userID uuid.UUID, photoKey string) error
	GetDetailWithRelations(ctx context.Context, userID uuid.UUID) (*model.Person, error)
	FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]model.Person, error)
}

type personRepository struct {
	db *gorm.DB
}

func NewPersonRepository(db *gorm.DB) PersonRepository {
	return &personRepository{db: db}
}

func (r *personRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Person, error) {
	var person model.Person
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&person).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPersonNotFound
		}
		return nil, err
	}
	return &person, nil
}

func (r *personRepository) FindByNIKHash(ctx context.Context, nikHash string) (*model.Person, error) {
	var person model.Person
	err := r.db.WithContext(ctx).Where("nik_hash = ?", nikHash).First(&person).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPersonNotFound
		}
		return nil, err
	}
	return &person, nil
}

func (r *personRepository) Save(ctx context.Context, person *model.Person) error {
	// Cek apakah NIK hash sudah dipakai oleh user_id yang BERBEDA
	var existing model.Person
	err := r.db.WithContext(ctx).Where("nik_hash = ? AND user_id != ?", person.NIKHash, person.UserID).First(&existing).Error
	if err == nil {
		return ErrNIKDuplicate
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Upsert person on conflict user_id
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"full_name", "nik", "nik_hash", "birth_place", "birth_date",
			"gender", "religion", "nationality", "phone", "updated_at",
		}),
	}).Save(person).Error
}

func (r *personRepository) UpdatePhoto(ctx context.Context, userID uuid.UUID, photoKey string) error {
	res := r.db.WithContext(ctx).Model(&model.Person{}).
		Where("user_id = ?", userID).
		Update("photo_object_key", photoKey)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPersonNotFound
	}
	return nil
}

func (r *personRepository) GetDetailWithRelations(ctx context.Context, userID uuid.UUID) (*model.Person, error) {
	var person model.Person
	err := r.db.WithContext(ctx).
		Preload("Addresses").
		Preload("Guardians").
		Preload("EducationHistories").
		Where("user_id = ?", userID).
		First(&person).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPersonNotFound
		}
		return nil, err
	}
	return &person, nil
}

func (r *personRepository) FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]model.Person, error) {
	var persons []model.Person
	err := r.db.WithContext(ctx).
		Preload("Addresses").
		Preload("Guardians").
		Preload("EducationHistories").
		Where("user_id IN ?", userIDs).
		Find(&persons).Error
	return persons, err
}
