package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrInstallmentRequestNotFound = errors.New("pengajuan cicilan tidak ditemukan")
	ErrInstallmentNotFound        = errors.New("termin cicilan tidak ditemukan")
)

type InstallmentRepository interface {
	CreateRequest(ctx context.Context, req *model.InstallmentRequest) error
	FindRequestByID(ctx context.Context, id uuid.UUID) (*model.InstallmentRequest, error)
	FindRequestByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*model.InstallmentRequest, error)
	ListRequests(ctx context.Context, status string) ([]model.InstallmentRequest, error)
	UpdateRequest(ctx context.Context, req *model.InstallmentRequest) error

	CreateInstallments(ctx context.Context, tx *gorm.DB, installments []model.Installment) error
	FindInstallmentsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]model.Installment, error)
	FindInstallmentByID(ctx context.Context, id uuid.UUID) (*model.Installment, error)
	UpdateInstallment(ctx context.Context, inst *model.Installment) error
	FindDueInstallments(ctx context.Context, startDate, endDate time.Time) ([]model.Installment, error)
	FindExpiredInstallments(ctx context.Context, now time.Time) ([]model.Installment, error)
}

type installmentRepository struct {
	db *gorm.DB
}

func NewInstallmentRepository(db *gorm.DB) InstallmentRepository {
	return &installmentRepository{db: db}
}

func (r *installmentRepository) CreateRequest(ctx context.Context, req *model.InstallmentRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *installmentRepository) FindRequestByID(ctx context.Context, id uuid.UUID) (*model.InstallmentRequest, error) {
	var req model.InstallmentRequest
	err := r.db.WithContext(ctx).
		Preload("Invoice").
		First(&req, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallmentRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *installmentRepository) FindRequestByInvoiceID(ctx context.Context, invoiceID uuid.UUID) (*model.InstallmentRequest, error) {
	var req model.InstallmentRequest
	err := r.db.WithContext(ctx).
		Preload("Invoice").
		Order("created_at desc").
		First(&req, "invoice_id = ?", invoiceID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *installmentRepository) ListRequests(ctx context.Context, status string) ([]model.InstallmentRequest, error) {
	var list []model.InstallmentRequest
	query := r.db.WithContext(ctx).Preload("Invoice").Order("created_at desc")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *installmentRepository) UpdateRequest(ctx context.Context, req *model.InstallmentRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *installmentRepository) CreateInstallments(ctx context.Context, tx *gorm.DB, installments []model.Installment) error {
	dbRunner := r.db
	if tx != nil {
		dbRunner = tx
	}
	return dbRunner.WithContext(ctx).Create(&installments).Error
}

func (r *installmentRepository) FindInstallmentsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]model.Installment, error) {
	var list []model.Installment
	err := r.db.WithContext(ctx).
		Where("invoice_id = ?", invoiceID).
		Order("term_no asc").
		Find(&list).Error
	return list, err
}

func (r *installmentRepository) FindInstallmentByID(ctx context.Context, id uuid.UUID) (*model.Installment, error) {
	var inst model.Installment
	err := r.db.WithContext(ctx).First(&inst, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstallmentNotFound
		}
		return nil, err
	}
	return &inst, nil
}

func (r *installmentRepository) UpdateInstallment(ctx context.Context, inst *model.Installment) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

func (r *installmentRepository) FindDueInstallments(ctx context.Context, startDate, endDate time.Time) ([]model.Installment, error) {
	var list []model.Installment
	err := r.db.WithContext(ctx).
		Where("status = ? AND due_date BETWEEN ? AND ?", model.InstallmentStatusUnpaid, startDate, endDate).
		Order("due_date asc").
		Find(&list).Error
	return list, err
}

func (r *installmentRepository) FindExpiredInstallments(ctx context.Context, now time.Time) ([]model.Installment, error) {
	var list []model.Installment
	err := r.db.WithContext(ctx).
		Where("status = ? AND due_date < ?", model.InstallmentStatusUnpaid, now).
		Find(&list).Error
	return list, err
}
