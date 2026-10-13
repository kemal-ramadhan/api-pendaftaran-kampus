package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/model"
	"gorm.io/gorm"
)

type ReferralRepository interface {
	// Codes
	CreateCode(ctx context.Context, code *model.ReferralCode) error
	GetCodeByID(ctx context.Context, id uuid.UUID) (*model.ReferralCode, error)
	GetCodeByCode(ctx context.Context, code string) (*model.ReferralCode, error)
	ListCodes(ctx context.Context, q, status string) ([]model.ReferralCode, error)
	ListCodesByPIC(ctx context.Context, userID uuid.UUID) ([]model.ReferralCode, error)
	UpdateCode(ctx context.Context, code *model.ReferralCode) error

	// PICs
	AddPIC(ctx context.Context, pic *model.ReferralCodePIC) error
	RemovePIC(ctx context.Context, codeID, userID uuid.UUID) error
	GetPICsByCode(ctx context.Context, codeID uuid.UUID) ([]model.ReferralCodePIC, error)

	// Usages
	CreateUsage(ctx context.Context, usage *model.ReferralUsage) error
	GetUsageByApplication(ctx context.Context, appID uuid.UUID) (*model.ReferralUsage, error)
	GetUsageByID(ctx context.Context, id uuid.UUID) (*model.ReferralUsage, error)
	DeleteUsage(ctx context.Context, id uuid.UUID) error
	UpdateUsage(ctx context.Context, usage *model.ReferralUsage) error
	ListUsagesByCode(ctx context.Context, codeID uuid.UUID) ([]model.ReferralUsage, error)

	// Transactions
	CreateTransaction(ctx context.Context, tx *model.ReferralTransaction) error
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*model.ReferralTransaction, error)
	GetTransactionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*model.ReferralTransaction, error)
	ListTransactionsByCode(ctx context.Context, codeID uuid.UUID, approveStatus, campusStatus string) ([]model.ReferralTransaction, error)
	UpdateTransaction(ctx context.Context, tx *model.ReferralTransaction) error

	// Settlements
	CreateSettlement(ctx context.Context, st *model.ReferralSettlement) error
	GetSettlementByID(ctx context.Context, id uuid.UUID) (*model.ReferralSettlement, error)
	ListSettlements(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralSettlement, error)
	UpdateSettlement(ctx context.Context, st *model.ReferralSettlement) error

	// Commission Payouts
	CreatePayout(ctx context.Context, p *model.ReferralCommissionPayout) error
	ListPayouts(ctx context.Context, codeID *uuid.UUID) ([]model.ReferralCommissionPayout, error)

	// Settings
	GetSettings(ctx context.Context) (*model.ReferralSetting, error)
	UpdateSettings(ctx context.Context, s *model.ReferralSetting) error
}

type referralRepository struct {
	db *gorm.DB
}

func NewReferralRepository(db *gorm.DB) ReferralRepository {
	return &referralRepository{db: db}
}

func (r *referralRepository) CreateCode(ctx context.Context, code *model.ReferralCode) error {
	code.Code = strings.ToUpper(strings.TrimSpace(code.Code))
	if code.PaymentCode == "" {
		code.PaymentCode = code.Code
	}
	return r.db.WithContext(ctx).Create(code).Error
}

func (r *referralRepository) GetCodeByID(ctx context.Context, id uuid.UUID) (*model.ReferralCode, error) {
	var c model.ReferralCode
	err := r.db.WithContext(ctx).Preload("PICs").Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *referralRepository) GetCodeByCode(ctx context.Context, code string) (*model.ReferralCode, error) {
	var c model.ReferralCode
	upperCode := strings.ToUpper(strings.TrimSpace(code))
	err := r.db.WithContext(ctx).Preload("PICs").Where("UPPER(code) = ?", upperCode).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *referralRepository) ListCodes(ctx context.Context, q, status string) ([]model.ReferralCode, error) {
	var list []model.ReferralCode
	query := r.db.WithContext(ctx).Preload("PICs").Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if q != "" {
		pattern := "%" + strings.ToLower(q) + "%"
		query = query.Where("LOWER(code) LIKE ? OR LOWER(institution_name) LIKE ?", pattern, pattern)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *referralRepository) ListCodesByPIC(ctx context.Context, userID uuid.UUID) ([]model.ReferralCode, error) {
	var list []model.ReferralCode
	err := r.db.WithContext(ctx).
		Preload("PICs").
		Joins("JOIN referral_code_pics ON referral_code_pics.code_id = referral_codes.id").
		Where("referral_code_pics.user_id = ?", userID).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

func (r *referralRepository) UpdateCode(ctx context.Context, code *model.ReferralCode) error {
	return r.db.WithContext(ctx).Save(code).Error
}

func (r *referralRepository) AddPIC(ctx context.Context, pic *model.ReferralCodePIC) error {
	return r.db.WithContext(ctx).Save(pic).Error
}

func (r *referralRepository) RemovePIC(ctx context.Context, codeID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("code_id = ? AND user_id = ?", codeID, userID).
		Delete(&model.ReferralCodePIC{}).Error
}

func (r *referralRepository) GetPICsByCode(ctx context.Context, codeID uuid.UUID) ([]model.ReferralCodePIC, error) {
	var pics []model.ReferralCodePIC
	err := r.db.WithContext(ctx).Where("code_id = ?", codeID).Find(&pics).Error
	return pics, err
}

func (r *referralRepository) CreateUsage(ctx context.Context, usage *model.ReferralUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

func (r *referralRepository) GetUsageByApplication(ctx context.Context, appID uuid.UUID) (*model.ReferralUsage, error) {
	var u model.ReferralUsage
	err := r.db.WithContext(ctx).Preload("ReferralCode").Where("application_id = ?", appID).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *referralRepository) GetUsageByID(ctx context.Context, id uuid.UUID) (*model.ReferralUsage, error) {
	var u model.ReferralUsage
	err := r.db.WithContext(ctx).Preload("ReferralCode").Where("id = ?", id).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *referralRepository) DeleteUsage(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ReferralUsage{}).Error
}

func (r *referralRepository) UpdateUsage(ctx context.Context, usage *model.ReferralUsage) error {
	return r.db.WithContext(ctx).Save(usage).Error
}

func (r *referralRepository) ListUsagesByCode(ctx context.Context, codeID uuid.UUID) ([]model.ReferralUsage, error) {
	var list []model.ReferralUsage
	err := r.db.WithContext(ctx).Where("code_id = ?", codeID).Order("attached_at DESC").Find(&list).Error
	return list, err
}

func (r *referralRepository) CreateTransaction(ctx context.Context, tx *model.ReferralTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *referralRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*model.ReferralTransaction, error) {
	var t model.ReferralTransaction
	err := r.db.WithContext(ctx).Preload("ReferralCode").Preload("ReferralUsage").Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *referralRepository) GetTransactionByInvoice(ctx context.Context, invoiceID uuid.UUID) (*model.ReferralTransaction, error) {
	var t model.ReferralTransaction
	err := r.db.WithContext(ctx).Preload("ReferralCode").Preload("ReferralUsage").Where("invoice_id = ?", invoiceID).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *referralRepository) ListTransactionsByCode(ctx context.Context, codeID uuid.UUID, approveStatus, campusStatus string) ([]model.ReferralTransaction, error) {
	var list []model.ReferralTransaction
	query := r.db.WithContext(ctx).Where("code_id = ?", codeID)
	if approveStatus != "" {
		query = query.Where("approve_status = ?", approveStatus)
	}
	if campusStatus != "" {
		query = query.Where("campus_status = ?", campusStatus)
	}
	err := query.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (r *referralRepository) UpdateTransaction(ctx context.Context, tx *model.ReferralTransaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

func (r *referralRepository) CreateSettlement(ctx context.Context, st *model.ReferralSettlement) error {
	return r.db.WithContext(ctx).Create(st).Error
}

func (r *referralRepository) GetSettlementByID(ctx context.Context, id uuid.UUID) (*model.ReferralSettlement, error) {
	var s model.ReferralSettlement
	err := r.db.WithContext(ctx).Preload("ReferralCode").Where("id = ?", id).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *referralRepository) ListSettlements(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralSettlement, error) {
	var list []model.ReferralSettlement
	query := r.db.WithContext(ctx).Preload("ReferralCode").Order("created_at DESC")
	if codeID != nil {
		query = query.Where("code_id = ?", *codeID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *referralRepository) UpdateSettlement(ctx context.Context, st *model.ReferralSettlement) error {
	return r.db.WithContext(ctx).Save(st).Error
}

func (r *referralRepository) CreatePayout(ctx context.Context, p *model.ReferralCommissionPayout) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *referralRepository) ListPayouts(ctx context.Context, codeID *uuid.UUID) ([]model.ReferralCommissionPayout, error) {
	var list []model.ReferralCommissionPayout
	query := r.db.WithContext(ctx).Order("paid_at DESC")
	if codeID != nil {
		query = query.Where("code_id = ?", *codeID)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *referralRepository) GetSettings(ctx context.Context) (*model.ReferralSetting, error) {
	var s model.ReferralSetting
	err := r.db.WithContext(ctx).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		defaultS := model.ReferralSetting{
			CommissionSettlementMode: "terpisah",
			DefaultCommissionType:    "persen",
			DefaultCommissionValue:   10,
			CreatedAt:                time.Now(),
			UpdatedAt:                time.Now(),
		}
		_ = r.db.WithContext(ctx).Create(&defaultS)
		return &defaultS, nil
	}
	return &s, err
}

func (r *referralRepository) UpdateSettings(ctx context.Context, s *model.ReferralSetting) error {
	s.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(s).Error
}
