package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindUsers(ctx context.Context, role, status, q string, page, perPage int) ([]model.User, int64, error)
	FindBatch(ctx context.Context, ids []uuid.UUID) ([]model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Preload("UserRoles.Role").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	identifier := strings.ToLower(strings.TrimSpace(email))
	if identifier == "" {
		return nil, nil
	}

	query := r.db.WithContext(ctx).Preload("UserRoles.Role")

	// 1. Jika mengandung '@', cari prioritas berdasarkan email pasti
	if strings.Contains(identifier, "@") {
		err := query.Where("LOWER(email) = ?", identifier).First(&user).Error
		if err == nil {
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 2. Pencocokan identitas persis: Email, Full Name, Username (bagian depan @), Nomor HP
	cleanDigits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, identifier)

	err := query.Where(
		`LOWER(email) = ? 
		OR LOWER(full_name) = ? 
		OR SPLIT_PART(LOWER(email), '@', 1) = ? 
		OR phone = ? 
		OR (length(?) >= 8 AND REGEXP_REPLACE(phone, '[^0-9]', '', 'g') = ?)`,
		identifier, identifier, identifier, identifier, cleanDigits, cleanDigits,
	).First(&user).Error

	if err == nil {
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 3. Alias sistem khusus (admin / superadmin / administrator)
	if identifier == "admin" || identifier == "superadmin" || identifier == "administrator" || identifier == "admin pmb" {
		err = r.db.WithContext(ctx).Preload("UserRoles.Role").
			Where("LOWER(email) = ? OR LOWER(email) = ?", "superadmin@kampus.ac.id", "superadmin@bth.id").
			First(&user).Error
		if err == nil {
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 4. Pencocokan nama depan / kata dalam full_name (misal: "Ujang" -> "Ujang Mahasiswa", "Rizky" -> "Muhammad Rizky Pratama")
	if len(identifier) >= 3 {
		err = r.db.WithContext(ctx).Preload("UserRoles.Role").
			Where("LOWER(full_name) ILIKE ?", "%"+identifier+"%").
			First(&user).Error
		if err == nil {
			return &user, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	return nil, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hash":        passwordHash,
			"password_changed_at": &now,
			"updated_at":           now,
		}).Error
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("last_login_at", &now).Error
}

func (r *userRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("status", status).Error
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	email = strings.ToLower(strings.TrimSpace(email))
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("email = ?", email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *userRepository) FindUsers(ctx context.Context, role, status, q string, page, perPage int) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	query := r.db.WithContext(ctx).Model(&model.User{})

	if status != "" {
		query = query.Where("users.status = ?", status)
	}

	if q != "" {
		search := "%" + strings.ToLower(q) + "%"
		query = query.Where("LOWER(users.full_name) LIKE ? OR LOWER(users.email) LIKE ?", search, search)
	}

	if role != "" {
		query = query.Joins("JOIN user_roles ON user_roles.user_id = users.id").
			Joins("JOIN roles ON roles.id = user_roles.role_id").
			Where("roles.code = ?", role)
	}

	var total int64
	if err := query.Distinct("users.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	err := query.Distinct("users.id", "users.email", "users.full_name", "users.phone", "users.status", "users.created_at", "users.updated_at").
		Preload("UserRoles.Role").
		Order("users.created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

func (r *userRepository) FindBatch(ctx context.Context, ids []uuid.UUID) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Preload("UserRoles.Role").
		Where("id IN ?", ids).
		Find(&users).Error
	return users, err
}

