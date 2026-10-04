package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/finance-service/internal/repository"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var (
	ErrDuplicateBankReference = errors.New("DUPLICATE_BANK_REFERENCE: nomor referensi mutasi bank telah digunakan pada transaksi lain")
	ErrInvalidAmount          = errors.New("INVALID_AMOUNT: nominal konfirmasi tidak sesuai dengan sisa tagihan")
	ErrRenewalLimitExceeded   = errors.New("RENEWAL_LIMIT_EXCEEDED: batas perpanjangan tagihan telah tercapai (maksimal 2 kali)")
)

// AddBusinessDays menghitung penambahan n hari kerja (Sabtu & Minggu dilewati)
func AddBusinessDays(start time.Time, n int) time.Time {
	cur := start
	added := 0
	for added < n {
		cur = cur.AddDate(0, 0, 1)
		if cur.Weekday() != time.Saturday && cur.Weekday() != time.Sunday {
			added++
		}
	}
	return cur
}

type FinanceAdminService interface {
	GetReconciliation(ctx context.Context, periodID *uint, status, channel, feeType, search string, page, limit int) (*ReconciliationResponse, error)
	ExportReconciliation(ctx context.Context, periodID *uint, format string) ([]byte, string, error)
	ManualConfirmPayment(ctx context.Context, invoiceID uuid.UUID, actorID uuid.UUID, actorName string, req ManualConfirmRequest) (*model.Payment, error)
	GetLedger(ctx context.Context, periodID *uint, channel string, from, to *time.Time, page, limit int) (*LedgerResponse, error)
	ExportLedger(ctx context.Context, periodID *uint, channel string, from, to *time.Time, format string) ([]byte, string, error)
	GetPaymentMethodsGrouped(ctx context.Context, amount int64) (*PaymentMethodsGroupedResponse, error)
	GetReceiptView(ctx context.Context, invoiceID uuid.UUID, userID uuid.UUID) (*ReceiptViewResponse, error)
	RenewInvoiceDueDate(ctx context.Context, invoiceID uuid.UUID, userID uuid.UUID) (*model.Invoice, error)
}

type financeAdminService struct {
	invoiceRepo  repository.InvoiceRepository
	paymentRepo  repository.PaymentRepository
	receiptRepo  repository.ReceiptRepository
	admissionCli internalclient.Client
	notifCli     internalclient.Client
}

func NewFinanceAdminService(
	invoiceRepo repository.InvoiceRepository,
	paymentRepo repository.PaymentRepository,
	receiptRepo repository.ReceiptRepository,
	admissionCli internalclient.Client,
	notifCli internalclient.Client,
) FinanceAdminService {
	return &financeAdminService{
		invoiceRepo:  invoiceRepo,
		paymentRepo:  paymentRepo,
		receiptRepo:  receiptRepo,
		admissionCli: admissionCli,
		notifCli:     notifCli,
	}
}

// GetReconciliation mengembalikan ringkasan rekonsiliasi dan daftar tagihan (PRD 2.3)
func (s *financeAdminService) GetReconciliation(ctx context.Context, periodID *uint, status, channel, feeType, search string, page, limit int) (*ReconciliationResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	invoices, total, err := s.invoiceRepo.ListAllInvoices(ctx, periodID, status, feeType, search, limit, offset)
	if err != nil {
		return nil, err
	}

	allInvoices, _, _ := s.invoiceRepo.ListAllInvoices(ctx, periodID, "", "", "", 10000, 0)

	var paidForms int64 = 0
	var totalForms int64 = 0
	var pendingManual int64 = 0
	var paidTuition int64 = 0
	var totalTuition int64 = 0
	var tuitionAmount int64 = 0
	var formAmount int64 = 0

	channelCounts := make(map[string]int64)
	channelTotals := make(map[string]int64)

	for _, inv := range allInvoices {
		if inv.FeeType == "registration" {
			totalForms++
			if inv.Status == model.InvoiceStatusPaid {
				paidForms++
				formAmount += inv.Amount
			}
		} else if inv.FeeType == "tuition" || inv.FeeType == "reregistration" {
			totalTuition++
			if inv.Status == model.InvoiceStatusPaid {
				paidTuition++
				tuitionAmount += inv.PaidAmount
			}
		}

		for _, p := range inv.Payments {
			if p.Status == model.PaymentStatusAwaitingVerification {
				pendingManual++
			}
			if p.Status == model.PaymentStatusSuccess {
				ch := p.Channel
				if ch == "" {
					ch = string(p.Method)
				}
				channelCounts[ch]++
				channelTotals[ch] += p.Amount
			}
		}
	}

	var paidFormsPct float64 = 0
	if totalForms > 0 {
		paidFormsPct = float64(paidForms) / float64(totalForms) * 100.0
	}

	// Persentase per kanal
	var totalSuccessfulPayments int64 = 0
	for _, cnt := range channelCounts {
		totalSuccessfulPayments += cnt
	}
	channelPcts := make(map[string]float64)
	var channelDists []ChannelDistributionItem
	for ch, cnt := range channelCounts {
		pct := 0.0
		if totalSuccessfulPayments > 0 {
			pct = float64(cnt) / float64(totalSuccessfulPayments) * 100.0
		}
		channelPcts[ch] = pct
		channelDists = append(channelDists, ChannelDistributionItem{
			Channel:    ch,
			Count:      cnt,
			Total:      channelTotals[ch],
			Percentage: pct,
		})
	}

	var items []ReconciliationInvoiceItem
	for _, inv := range invoices {
		ch := "-"
		var paidAt *time.Time
		for _, p := range inv.Payments {
			if p.Status == model.PaymentStatusSuccess {
				ch = p.Channel
				if ch == "" {
					ch = string(p.Method)
				}
				paidAt = p.PaidAt
				break
			}
		}

		stLabel := mapInvoiceStatusLabel(inv.Status)
		if inv.PaymentScheme == "free" || (inv.Amount == 0 && inv.Status == model.InvoiceStatusPaid) {
			stLabel = "Gratis"
		}
		items = append(items, ReconciliationInvoiceItem{
			ID:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			FullName:      fmt.Sprintf("Pendaftar %s", inv.InvoiceNumber),
			ProgramName:   "Program Studi",
			FeeType:       inv.FeeType,
			Amount:        inv.Amount,
			PaidAmount:    inv.PaidAmount,
			Channel:       ch,
			Status:        inv.Status,
			StatusLabel:   stLabel,
			DueDate:       inv.DueDate,
			PaidAt:        paidAt,
		})
	}

	res := &ReconciliationResponse{
		PeriodID: periodID,
		KPIs: ReconciliationKPIs{
			PaidFormsCount:      paidForms,
			TotalFormsCount:     totalForms,
			PaidFormsPercentage: paidFormsPct,
			PendingManualCount:  pendingManual,
			PaidTuitionCount:    paidTuition,
			TotalTuitionCount:   totalTuition,
			TotalTuitionAmount:  tuitionAmount,
			ChannelPercentages:  channelPcts,
		},
		FundProportions: FundProportions{
			FormAmount:    formAmount,
			TuitionAmount: tuitionAmount,
		},
		ChannelDistributions: channelDists,
		Items:                items,
	}
	res.Meta.Total = total
	res.Meta.Page = page
	res.Meta.Limit = limit

	return res, nil
}

// ExportReconciliation mengekspor data rekonsiliasi ke CSV atau Excel
func (s *financeAdminService) ExportReconciliation(ctx context.Context, periodID *uint, format string) ([]byte, string, error) {
	invoices, _, err := s.invoiceRepo.ListAllInvoices(ctx, periodID, "", "", "", 10000, 0)
	if err != nil {
		return nil, "", err
	}

	if strings.ToLower(format) == "csv" {
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		_ = writer.Write([]string{"No", "Nomor Tagihan", "Jenis Biaya", "Nominal", "Terbayar", "Status", "Jatuh Tempo"})
		for i, inv := range invoices {
			_ = writer.Write([]string{
				fmt.Sprintf("%d", i+1),
				inv.InvoiceNumber,
				inv.FeeType,
				fmt.Sprintf("%d", inv.Amount),
				fmt.Sprintf("%d", inv.PaidAmount),
				string(inv.Status),
				inv.DueDate.Format("2006-01-02"),
			})
		}
		writer.Flush()
		return buf.Bytes(), "rekonsiliasi-keuangan.csv", nil
	}

	// XLSX
	f := excelize.NewFile()
	sheetName := "Rekonsiliasi"
	f.SetSheetName("Sheet1", sheetName)
	_ = f.SetSheetRow(sheetName, "A1", &[]interface{}{"No", "Nomor Tagihan", "Jenis Biaya", "Nominal", "Terbayar", "Status", "Jatuh Tempo"})
	for i, inv := range invoices {
		row := i + 2
		_ = f.SetSheetRow(sheetName, fmt.Sprintf("A%d", row), &[]interface{}{
			i + 1,
			inv.InvoiceNumber,
			inv.FeeType,
			inv.Amount,
			inv.PaidAmount,
			string(inv.Status),
			inv.DueDate.Format("2006-01-02"),
		})
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "rekonsiliasi-keuangan.xlsx", nil
}

// ManualConfirmPayment memproses konfirmasi lunas manual oleh admin/keuangan (PRD 2.3)
func (s *financeAdminService) ManualConfirmPayment(ctx context.Context, invoiceID uuid.UUID, actorID uuid.UUID, actorName string, req ManualConfirmRequest) (*model.Payment, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	// 1. Pengaman: Tolak jika sudah lunas
	if inv.IsPaid() {
		return nil, ErrInvoiceAlreadyPaid
	}

	// 2. Pengaman: Tolak jika nominal tidak cocok dengan sisa tagihan
	remaining := inv.Amount - inv.PaidAmount
	if req.Amount != remaining {
		return nil, fmt.Errorf("%w: nominal tagihan Rp %d, nominal dikirim Rp %d", ErrInvalidAmount, remaining, req.Amount)
	}

	// 3. Pengaman: Tolak jika nomor referensi mutasi bank sudah pernah dipakai
	if req.BankReference != "" {
		existing, _ := s.paymentRepo.FindByBankReference(ctx, req.BankReference)
		if existing != nil {
			return nil, ErrDuplicateBankReference
		}
	}

	now := time.Now()
	txTime := req.TransactionTime
	if txTime.IsZero() {
		txTime = now
	}

	payment := &model.Payment{
		ID:             uuid.New(),
		InvoiceID:      inv.ID,
		Method:         model.PaymentMethodManualConfirmation,
		Provider:       "manual",
		ProviderRef:    fmt.Sprintf("MANUAL-%s-%s", inv.InvoiceNumber, req.BankReference),
		BankReference:  req.BankReference,
		Channel:        req.Channel,
		Amount:         req.Amount,
		Status:         model.PaymentStatusSuccess,
		ProofURL:       req.ProofURL,
		Notes:          req.Reason,
		VerifiedBy:     &actorID,
		VerifiedAt:     &now,
		PaidAt:         &txTime,
		SettlementDate: &now,
	}

	receipt := &model.Receipt{
		ID:            uuid.New(),
		PaymentID:     payment.ID,
		InvoiceID:     inv.ID,
		ReceiptNumber: fmt.Sprintf("KWT-%d-%s", now.Year(), payment.ID.String()[:8]),
		CreatedAt:     now,
	}

	if s.invoiceRepo.DB() != nil {
		err = s.invoiceRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Batalkan pembayaran pending Duitku jika ada
			_ = tx.Model(&model.Payment{}).
				Where("invoice_id = ? AND status = ?", inv.ID, model.PaymentStatusPending).
				Update("status", model.PaymentStatusCancelled).Error

			// Buat pembayaran manual sukses
			if err := tx.Create(payment).Error; err != nil {
				return err
			}

			// Terbitkan kwitansi resmi
			if err := tx.Create(receipt).Error; err != nil {
				return err
			}
			payment.Receipt = receipt

			// Update invoice lunas
			return tx.Model(&model.Invoice{}).
				Where("id = ?", inv.ID).
				Updates(map[string]interface{}{
					"status":      model.InvoiceStatusPaid,
					"paid_amount": inv.Amount,
					"updated_at":  now,
				}).Error
		})
	} else {
		if err := s.paymentRepo.CreateInTx(nil, payment); err != nil {
			return nil, err
		}
		if s.receiptRepo != nil {
			_ = s.receiptRepo.CreateInTx(nil, receipt)
		}
		payment.Receipt = receipt
		err = s.invoiceRepo.UpdateStatus(ctx, inv.ID, model.InvoiceStatusPaid, inv.Amount)
	}

	if err != nil {
		return nil, err
	}

	// Update status di admission-service
	if inv.ApplicationID != nil && s.admissionCli != nil {
		go func() {
			payload := map[string]interface{}{
				"status": "form_open",
				"notes":  fmt.Sprintf("Pembayaran formulir lunas via konfirmasi manual (Ref: %s)", req.BankReference),
			}
			var resp interface{}
			_ = s.admissionCli.Post(context.Background(), fmt.Sprintf("/internal/applications/%s/status", inv.ApplicationID.String()), payload, &resp)
		}()
	}

	// Kirim event aktivitas admin ke notification-service (PRD 2.9)
	if s.notifCli != nil {
		go func() {
			actPayload := map[string]interface{}{
				"actor_id":     actorID.String(),
				"actor_name":   actorName,
				"actor_role":   "keuangan",
				"action":       "manual_confirm",
				"action_label": "Konfirmasi Lunas Manual",
				"target_type":  "invoice",
				"target_id":    inv.ID.String(),
				"target_label": fmt.Sprintf("Invoice %s", inv.InvoiceNumber),
				"service":      "finance-service",
			}
			var actResp interface{}
			_ = s.notifCli.Post(context.Background(), "/internal/activity", actPayload, &actResp)
		}()
	}

	return payment, nil
}

// GetLedger mengembalikan data buku besar kas masuk (PRD 2.7)
func (s *financeAdminService) GetLedger(ctx context.Context, periodID *uint, channel string, from, to *time.Time, page, limit int) (*LedgerResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	payments, err := s.paymentRepo.FindSuccessfulPayments(ctx, periodID, channel, from, to)
	if err != nil {
		return nil, err
	}

	var totalCredit int64 = 0
	var runningBalance int64 = 0
	channelTotals := make(map[string]int64)
	channelCounts := make(map[string]int64)

	var allItems []LedgerItem
	for _, p := range payments {
		if p.Method == model.PaymentMethodFree || p.Method == "free" || p.Amount == 0 {
			continue
		}
		credit := p.Amount
		totalCredit += credit
		runningBalance += credit

		ch := p.Channel
		if ch == "" {
			ch = string(p.Method)
		}
		channelTotals[ch] += credit
		channelCounts[ch]++

		srcLabel := "Duitku VA"
		if p.Method == model.PaymentMethodManualConfirmation {
			srcLabel = "Konfirmasi Manual Bank"
		} else if p.Method == model.PaymentMethodManualTransfer {
			srcLabel = "Transfer Bank Manual"
		} else if p.Method == model.PaymentMethodQRIS {
			srcLabel = "QRIS"
		}

		ref := p.BankReference
		if ref == "" {
			ref = p.ProviderRef
		}

		paidAt := p.CreatedAt
		if p.PaidAt != nil {
			paidAt = *p.PaidAt
		}

		allItems = append(allItems, LedgerItem{
			TransactionTime: paidAt,
			InvoiceNumber:   p.InvoiceID.String(),
			PayerName:       fmt.Sprintf("Pendaftar %s", p.InvoiceID.String()[:8]),
			FeeType:         "Formulir / UKT",
			Channel:         ch,
			ReferenceNumber: ref,
			CreditAmount:    credit,
			RunningBalance:  runningBalance,
			SourceLabel:     srcLabel,
		})
	}

	var summaryCards []LedgerSummaryCard
	for ch, tot := range channelTotals {
		summaryCards = append(summaryCards, LedgerSummaryCard{
			Channel:      ch,
			TotalAmount:  tot,
			TotalLabel:   fmt.Sprintf("Rp %d", tot),
			CountRecords: channelCounts[ch],
		})
	}

	totalRecords := int64(len(allItems))
	start := (page - 1) * limit
	end := start + limit
	if start > len(allItems) {
		start = len(allItems)
	}
	if end > len(allItems) {
		end = len(allItems)
	}

	pagedItems := allItems[start:end]

	res := &LedgerResponse{
		PeriodID:     periodID,
		SummaryCards: summaryCards,
		Items:        pagedItems,
		TotalCredit:  totalCredit,
	}
	res.Meta.Total = totalRecords
	res.Meta.Page = page
	res.Meta.Limit = limit

	return res, nil
}

// ExportLedger mengekspor buku besar ke CSV atau Excel
func (s *financeAdminService) ExportLedger(ctx context.Context, periodID *uint, channel string, from, to *time.Time, format string) ([]byte, string, error) {
	ledger, err := s.GetLedger(ctx, periodID, channel, from, to, 1, 100000)
	if err != nil {
		return nil, "", err
	}

	if strings.ToLower(format) == "csv" {
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		_ = writer.Write([]string{"Waktu Transaksi", "Nomor Tagihan", "Nama Pembayar", "Kanal", "Referensi", "Kredit Masuk", "Saldo Berjalan", "Sumber"})
		for _, item := range ledger.Items {
			_ = writer.Write([]string{
				item.TransactionTime.Format("2006-01-02 15:04:05"),
				item.InvoiceNumber,
				item.PayerName,
				item.Channel,
				item.ReferenceNumber,
				fmt.Sprintf("%d", item.CreditAmount),
				fmt.Sprintf("%d", item.RunningBalance),
				item.SourceLabel,
			})
		}
		writer.Flush()
		return buf.Bytes(), "buku-besar-kas.csv", nil
	}

	// XLSX
	f := excelize.NewFile()
	sheetName := "BukuBesar"
	f.SetSheetName("Sheet1", sheetName)
	_ = f.SetSheetRow(sheetName, "A1", &[]interface{}{"Waktu Transaksi", "Nomor Tagihan", "Nama Pembayar", "Kanal", "Referensi", "Kredit Masuk", "Saldo Berjalan", "Sumber"})
	for i, item := range ledger.Items {
		row := i + 2
		_ = f.SetSheetRow(sheetName, fmt.Sprintf("A%d", row), &[]interface{}{
			item.TransactionTime.Format("2006-01-02 15:04:05"),
			item.InvoiceNumber,
			item.PayerName,
			item.Channel,
			item.ReferenceNumber,
			item.CreditAmount,
			item.RunningBalance,
			item.SourceLabel,
		})
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "buku-besar-kas.xlsx", nil
}

// GetPaymentMethodsGrouped mengembalikan daftar metode pembayaran dikelompokkan per tab (PRD 3.5)
func (s *financeAdminService) GetPaymentMethodsGrouped(ctx context.Context, amount int64) (*PaymentMethodsGroupedResponse, error) {
	vaList := []PaymentMethodTabItem{
		{
			Group:      "virtual_account",
			Code:       "VA_BSI",
			Name:       "Bank Syariah Indonesia (BSI)",
			BankName:   "BSI",
			BadgeLabel: "Bank Utama BTH",
			Fee:        0,
			FeeLabel:   "Biaya Transaksi Rp 0 (Gratis)",
		},
		{
			Group:      "virtual_account",
			Code:       "VA_MANDIRI",
			Name:       "Bank Mandiri Virtual Account",
			BankName:   "Mandiri",
			BadgeLabel: "Bank Mitra",
			Fee:        0,
			FeeLabel:   "Biaya Transaksi Rp 0 (Gratis)",
		},
		{
			Group:      "virtual_account",
			Code:       "VA_BNI",
			Name:       "Bank BNI Virtual Account",
			BankName:   "BNI",
			BadgeLabel: "Bank Mitra",
			Fee:        0,
			FeeLabel:   "Biaya Transaksi Rp 0 (Gratis)",
		},
	}

	transferList := []PaymentMethodTabItem{
		{
			Group:         "bank_transfer",
			Code:          "TRANSFER_BTH",
			Name:          "Transfer Manual Bank BTH",
			BankName:      "BSI Cabang Tasikmalaya",
			BadgeLabel:    "Rekening Resmi Kampus",
			Fee:           0,
			FeeLabel:      "Biaya Transaksi Rp 0 (Gratis)",
			AccountNumber: "7001234567",
			AccountName:   "Universitas Bakti Tunas Husada",
		},
	}

	qrisList := []PaymentMethodTabItem{
		{
			Group:      "qris",
			Code:       "QRIS",
			Name:       "QRIS Duitku (Semua Dompet Digital & Bank)",
			BankName:   "QRIS",
			BadgeLabel: "Instant Verification",
			Fee:        0,
			FeeLabel:   "Biaya Transaksi Rp 0 (Gratis)",
			PaymentURL: "https://passport.duitku.com/qris-mock",
			QRString:   "00020101021226670016ID.CO.DUITKU.WWW01189360099900000000000215988202601944615204581253033605802ID5925UNIVERSITAS BAKTI TUNAS HUSADA6011TASIKMALAYA61054611562070703A016304E2B1",
		},
	}

	notes := []string{
		"Verifikasi pembayaran berlangsung secara otomatis 24 jam melalui Virtual Account & QRIS.",
		"Setelah pembayaran terverifikasi lunas, modul Ujian CBT Online dan unggah berkas langsung dibuka.",
		"Biaya pendaftaran yang telah dibayarkan tidak dapat ditarik kembali.",
	}

	return &PaymentMethodsGroupedResponse{
		VirtualAccount: vaList,
		BankTransfer:   transferList,
		QRIS:           qrisList,
		Notes:          notes,
	}, nil
}

// GetReceiptView mengembalikan tampilan kuitansi resmi pelunasan (PRD 3.8)
func (s *financeAdminService) GetReceiptView(ctx context.Context, invoiceID uuid.UUID, userID uuid.UUID) (*ReceiptViewResponse, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	var paidAt *time.Time
	methodLabel := "Virtual Account Bank BSI"
	for _, p := range inv.Payments {
		if p.Status == model.PaymentStatusSuccess {
			paidAt = p.PaidAt
			if p.Method == model.PaymentMethodManualConfirmation {
				methodLabel = fmt.Sprintf("Transfer Bank (%s)", p.Channel)
			} else if p.Method == model.PaymentMethodQRIS {
				methodLabel = "QRIS Duitku"
			}
			break
		}
	}

	return &ReceiptViewResponse{
		Title:              "Pembayaran Formulir Terverifikasi Lunas",
		InvoiceNumber:      inv.InvoiceNumber,
		FullName:           fmt.Sprintf("Pendaftar %s", inv.InvoiceNumber),
		RegistrationNumber: fmt.Sprintf("REG-%s", inv.InvoiceNumber[len(inv.InvoiceNumber)-4:]),
		ProgramName:        "Pilihan Program Studi",
		PaymentMethodLabel: methodLabel,
		VerifiedAt:         paidAt,
		TotalAmount:        inv.PaidAmount,
		TotalAmountLabel:   fmt.Sprintf("Rp %d", inv.PaidAmount),
		LockNoticeLabel:    "Pembayaran lunas. Pilihan jalur dan program studi terkunci; perubahan hanya melalui pengajuan ke panitia.",
		NextAction: &ReceiptNextAction{
			Title:       "Lanjut ke Ujian CBT Online",
			ActionURL:   "/exam",
			ButtonLabel: "Buka Ujian CBT",
		},
	}, nil
}

// RenewInvoiceDueDate memperpanjang batas pembayaran jatuh tempo 3 hari kerja (maksimal 2 kali) (PRD 3.5 & K15)
func (s *financeAdminService) RenewInvoiceDueDate(ctx context.Context, invoiceID uuid.UUID, userID uuid.UUID) (*model.Invoice, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	if inv.UserID != userID {
		return nil, ErrForbiddenAccess
	}

	if inv.PaymentScheme == "free" || inv.Amount == 0 {
		return nil, ErrInvoiceFree
	}

	if inv.IsPaid() {
		return nil, ErrInvoiceAlreadyPaid
	}

	if inv.RenewalCount >= 2 {
		return nil, ErrRenewalLimitExceeded
	}

	newDueDate := AddBusinessDays(time.Now(), 3)
	err = s.invoiceRepo.UpdateDueDateAndRenewal(ctx, inv.ID, newDueDate, inv.RenewalCount+1)
	if err != nil {
		return nil, err
	}

	return s.invoiceRepo.FindByID(ctx, inv.ID)
}

func mapInvoiceStatusLabel(st model.InvoiceStatus) string {
	switch st {
	case model.InvoiceStatusPaid:
		return "Lunas"
	case model.InvoiceStatusUnpaid:
		return "Belum Bayar"
	case model.InvoiceStatusPartiallyPaid:
		return "Sebagian Terbayar"
	case model.InvoiceStatusExpired:
		return "Kedaluwarsa"
	case model.InvoiceStatusCancelled:
		return "Dibatalkan"
	default:
		return string(st)
	}
}
