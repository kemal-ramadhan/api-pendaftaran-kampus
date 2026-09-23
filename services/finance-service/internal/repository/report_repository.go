package repository

import (
	"context"

	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"gorm.io/gorm"
)

type ReportRepository interface {
	GetSummary(ctx context.Context) (*model.FinancialReportSummaryResponse, error)
	GetAllInvoicesForExport(ctx context.Context) ([]model.Invoice, error)
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetSummary(ctx context.Context) (*model.FinancialReportSummaryResponse, error) {
	resp := &model.FinancialReportSummaryResponse{
		ByFeeType:          make([]model.FeeTypeSummaryItem, 0),
		RecentTransactions: make([]model.TransactionSummaryItem, 0),
	}

	// 1. Agregat Global
	type GlobalAgg struct {
		TotalInvoices int64 `gorm:"column:total_invoices"`
		TotalExpected int64 `gorm:"column:total_expected"`
		TotalPaid     int64 `gorm:"column:total_paid"`
	}
	var gAgg GlobalAgg
	err := r.db.WithContext(ctx).Model(&model.Invoice{}).
		Select("COUNT(*) as total_invoices, COALESCE(SUM(amount), 0) as total_expected, COALESCE(SUM(paid_amount), 0) as total_paid").
		Where("status != ?", model.InvoiceStatusCancelled).
		Scan(&gAgg).Error
	if err != nil {
		return nil, err
	}

	resp.TotalInvoices = gAgg.TotalInvoices
	resp.TotalExpectedAmount = gAgg.TotalExpected
	resp.TotalPaidAmount = gAgg.TotalPaid
	resp.TotalOutstandingAmount = gAgg.TotalExpected - gAgg.TotalPaid
	if resp.TotalOutstandingAmount < 0 {
		resp.TotalOutstandingAmount = 0
	}

	// 2. Agregat per FeeType
	type FeeTypeAgg struct {
		FeeType       string `gorm:"column:fee_type"`
		Count         int64  `gorm:"column:count"`
		TotalAmount   int64  `gorm:"column:total_amount"`
		PaidAmount    int64  `gorm:"column:paid_amount"`
	}
	var ftAggs []FeeTypeAgg
	err = r.db.WithContext(ctx).Model(&model.Invoice{}).
		Select("fee_type, COUNT(*) as count, COALESCE(SUM(amount), 0) as total_amount, COALESCE(SUM(paid_amount), 0) as paid_amount").
		Where("status != ?", model.InvoiceStatusCancelled).
		Group("fee_type").
		Scan(&ftAggs).Error
	if err != nil {
		return nil, err
	}

	for _, item := range ftAggs {
		outstanding := item.TotalAmount - item.PaidAmount
		if outstanding < 0 {
			outstanding = 0
		}
		resp.ByFeeType = append(resp.ByFeeType, model.FeeTypeSummaryItem{
			FeeType:           item.FeeType,
			Count:             item.Count,
			TotalAmount:       item.TotalAmount,
			PaidAmount:        item.PaidAmount,
			OutstandingAmount: outstanding,
		})
	}

	// 3. Transaksi Pembayaran Terbaru
	type RecentTxRow struct {
		PaymentID     string `gorm:"column:id"`
		InvoiceNumber string `gorm:"column:invoice_number"`
		FeeType       string `gorm:"column:fee_type"`
		Amount        int64  `gorm:"column:amount"`
		Method        string `gorm:"column:method"`
		Status        string `gorm:"column:status"`
		PaidAt        string `gorm:"column:paid_at"`
	}
	var recentRows []model.Payment
	err = r.db.WithContext(ctx).
		Preload("Receipt").
		Where("status = ?", model.PaymentStatusSuccess).
		Order("paid_at desc, created_at desc").
		Limit(10).
		Find(&recentRows).Error
	if err != nil {
		return nil, err
	}

	for _, p := range recentRows {
		var inv model.Invoice
		_ = r.db.WithContext(ctx).First(&inv, "id = ?", p.InvoiceID).Error

		paidTime := p.CreatedAt
		if p.PaidAt != nil {
			paidTime = *p.PaidAt
		}

		resp.RecentTransactions = append(resp.RecentTransactions, model.TransactionSummaryItem{
			PaymentID:     p.ID,
			InvoiceNumber: inv.InvoiceNumber,
			FeeType:       inv.FeeType,
			Amount:        p.Amount,
			Method:        string(p.Method),
			Status:        string(p.Status),
			PaidAt:        paidTime,
		})
	}

	return resp, nil
}

func (r *reportRepository) GetAllInvoicesForExport(ctx context.Context) ([]model.Invoice, error) {
	var invoices []model.Invoice
	err := r.db.WithContext(ctx).
		Preload("Payments.Receipt").
		Preload("Installments").
		Order("created_at desc").
		Find(&invoices).Error
	return invoices, err
}
