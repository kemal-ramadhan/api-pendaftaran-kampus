package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrReferralNotFound    = errors.New("kode referal tidak ditemukan atau tidak aktif")
	ErrReferralConsentReq  = errors.New("persetujuan konfirmasi pembayaran ke guru BK wajib disetujui")
	ErrReferralLocked      = errors.New("REFERRAL_LOCKED: kode referal sudah terverifikasi dan tidak dapat diubah")
	ErrInvoiceBilledPartner= errors.New("INVOICE_BILLED_TO_PARTNER: tagihan ditagihkan melalui guru BK")
)

type ReferralService interface {
	CheckCode(ctx context.Context, code string) (*dto.CheckCodeResponse, error)
	GetMyReferral(ctx context.Context, userID, applicationID uuid.UUID) (*model.ReferralUsage, error)
	AttachCode(ctx context.Context, userID, applicationID uuid.UUID, req dto.AttachReferralRequest) (*model.ReferralUsage, error)
	RemoveCode(ctx context.Context, userID, applicationID uuid.UUID) error

	// Partner (Guru BK)
	ListPartnerCodes(ctx context.Context, userID uuid.UUID) ([]model.ReferralCode, error)
	GetCodeDetail(ctx context.Context, codeID, userID uuid.UUID, isAdmin bool) (*model.ReferralCode, *dto.ReferralSummaryResponse, error)
	ListCodeUsers(ctx context.Context, codeID, userID uuid.UUID, isAdmin bool, q, approveStatus, campusStatus string) ([]dto.ReferralUserItem, error)
	ApproveTransactions(ctx context.Context, txIDs []uuid.UUID, actorID uuid.UUID, isAdmin bool) error
	RejectTransaction(ctx context.Context, txID uuid.UUID, reason string, actorID uuid.UUID, isAdmin bool) error
	SubmitSettlement(ctx context.Context, codeID, userID uuid.UUID, req dto.SubmitSettlementRequest) (*model.ReferralSettlement, error)
	ListSettlements(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralSettlement, error)
	VerifySettlement(ctx context.Context, settlementID, actorID uuid.UUID) error
	RejectSettlement(ctx context.Context, settlementID uuid.UUID, reason string, actorID uuid.UUID) error
	ListCommissions(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralTransaction, error)
	CreatePayout(ctx context.Context, req dto.CreateCommissionPayoutRequest, actorID uuid.UUID) error
	GetPartnerDashboard(ctx context.Context, userID uuid.UUID) (*dto.PartnerDashboardResponse, error)

	// Admin
	ListAdminCodes(ctx context.Context, q, status string) ([]model.ReferralCode, error)
	CreateCode(ctx context.Context, code *model.ReferralCode) error
	UpdateCode(ctx context.Context, code *model.ReferralCode) error
	ImportPICsCSV(ctx context.Context, r io.Reader, actorID uuid.UUID) (created int, updated int, failed int, errs []string, err error)
	GetSettings(ctx context.Context) (*model.ReferralSetting, error)
	UpdateSettings(ctx context.Context, s *model.ReferralSetting) error
	ChangeApplicationReferral(ctx context.Context, appID, newCodeID, actorID uuid.UUID, reason string) error
}

type referralService struct {
	repo    repository.ReferralRepository
	baseDB  *gorm.DB
	log     *zap.Logger
}

func NewReferralService(repo repository.ReferralRepository, baseDB *gorm.DB, log *zap.Logger) ReferralService {
	return &referralService{
		repo:   repo,
		baseDB: baseDB,
		log:    log,
	}
}

func MaskNIK(nik string) string {
	nik = strings.TrimSpace(nik)
	if len(nik) <= 8 {
		return "3204********0002"
	}
	prefix := nik[:4]
	suffix := nik[len(nik)-4:]
	return prefix + "********" + suffix
}

func (s *referralService) CheckCode(ctx context.Context, code string) (*dto.CheckCodeResponse, error) {
	c, err := s.repo.GetCodeByCode(ctx, code)
	if err != nil || c == nil || c.Status != "active" {
		return &dto.CheckCodeResponse{Valid: false}, nil
	}

	now := time.Now()
	if c.ValidFrom != nil && now.Before(*c.ValidFrom) {
		return &dto.CheckCodeResponse{Valid: false}, nil
	}
	if c.ValidUntil != nil && now.After(*c.ValidUntil) {
		return &dto.CheckCodeResponse{Valid: false}, nil
	}

	picName := ""
	if len(c.PICs) > 0 {
		var user struct {
			FullName string
		}
		_ = s.baseDB.Table("pmb_auth.users").Select("full_name").Where("id = ?", c.PICs[0].UserID).Scan(&user).Error
		picName = user.FullName
	}

	return &dto.CheckCodeResponse{
		Valid:           true,
		InstitutionName: c.InstitutionName,
		PICName:         picName,
	}, nil
}

func (s *referralService) GetMyReferral(ctx context.Context, userID, applicationID uuid.UUID) (*model.ReferralUsage, error) {
	return s.repo.GetUsageByApplication(ctx, applicationID)
}

func (s *referralService) AttachCode(ctx context.Context, userID, applicationID uuid.UUID, req dto.AttachReferralRequest) (*model.ReferralUsage, error) {
	if !req.Consent {
		return nil, ErrReferralConsentReq
	}

	c, err := s.repo.GetCodeByCode(ctx, req.Code)
	if err != nil || c == nil || c.Status != "active" {
		return nil, ErrReferralNotFound
	}

	existing, _ := s.repo.GetUsageByApplication(ctx, applicationID)
	if existing != nil {
		if existing.LockedAt != nil {
			return nil, ErrReferralLocked
		}
		existing.CodeID = c.ID
		now := time.Now()
		existing.ConsentAt = &now
		existing.AttachedAt = now
		_ = s.repo.UpdateUsage(ctx, existing)
		return existing, nil
	}

	now := time.Now()
	usage := model.ReferralUsage{
		ID:            uuid.New(),
		CodeID:        c.ID,
		ApplicationID: applicationID,
		UserID:        userID,
		ConsentAt:     &now,
		AttachedAt:    now,
	}

	if err := s.repo.CreateUsage(ctx, &usage); err != nil {
		return nil, err
	}

	// Cari tagihan pendaftaran camaba di skema finance
	var invoice struct {
		ID     uuid.UUID
		Amount float64
		Status string
	}
	err = s.baseDB.Table("finance.invoices").
		Where("user_id = ? AND fee_type = 'registration' AND status != 'paid'", userID).
		Order("created_at DESC").
		Limit(1).
		Scan(&invoice).Error

	if err == nil && invoice.ID != uuid.Nil {
		// Buat transaksi referal status PENDING
		tx := model.ReferralTransaction{
			ID:            uuid.New(),
			CodeID:        c.ID,
			UsageID:       usage.ID,
			InvoiceID:     invoice.ID,
			Type:          "REGISTRATION",
			Amount:        invoice.Amount,
			ApproveStatus: "PENDING",
			CampusStatus:  "TAGIHAN",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		_ = s.repo.CreateTransaction(ctx, &tx)
		// Tandai invoice sebagai ditagihkan ke mitra
		_ = s.baseDB.Table("finance.invoices").Where("id = ?", invoice.ID).Update("notes", "BILLED_TO_PARTNER:"+c.Code).Error
	}

	return &usage, nil
}

func (s *referralService) RemoveCode(ctx context.Context, userID, applicationID uuid.UUID) error {
	existing, err := s.repo.GetUsageByApplication(ctx, applicationID)
	if err != nil || existing == nil {
		return nil
	}
	if existing.LockedAt != nil {
		return ErrReferralLocked
	}
	return s.repo.DeleteUsage(ctx, existing.ID)
}

func (s *referralService) ListPartnerCodes(ctx context.Context, userID uuid.UUID) ([]model.ReferralCode, error) {
	return s.repo.ListCodesByPIC(ctx, userID)
}

func (s *referralService) GetCodeDetail(ctx context.Context, codeID, userID uuid.UUID, isAdmin bool) (*model.ReferralCode, *dto.ReferralSummaryResponse, error) {
	c, err := s.repo.GetCodeByID(ctx, codeID)
	if err != nil || c == nil {
		return nil, nil, ErrReferralNotFound
	}

	if !isAdmin {
		// Cek otorisasi PIC
		isOwner := false
		for _, pic := range c.PICs {
			if pic.UserID == userID {
				isOwner = true
				break
			}
		}
		if !isOwner {
			return nil, nil, ErrReferralNotFound
		}
	}

	summary, err := s.calculateSummary(ctx, codeID, c)
	if err != nil {
		return nil, nil, err
	}

	return c, summary, nil
}

func (s *referralService) calculateSummary(ctx context.Context, codeID uuid.UUID, c *model.ReferralCode) (*dto.ReferralSummaryResponse, error) {
	usages, _ := s.repo.ListUsagesByCode(ctx, codeID)
	totalUsers := int64(len(usages))

	txs, _ := s.repo.ListTransactionsByCode(ctx, codeID, "", "")
	settlements, _ := s.repo.ListSettlements(ctx, &codeID, "")

	var allSuccess float64
	var pendingQueue float64
	var totalCommission float64
	var paidCommission float64
	var unreturnedBills float64

	mode := c.CommissionSettlementMode
	if mode == "" {
		st, _ := s.repo.GetSettings(ctx)
		if st != nil {
			mode = st.CommissionSettlementMode
		}
	}

	for _, t := range txs {
		if t.ApproveStatus == "PENDING" {
			pendingQueue += t.Amount
		} else if t.ApproveStatus == "SUCCESS" {
			allSuccess += t.Amount
			totalCommission += t.CommissionAmount
			if t.CommissionStatus == "DIBAYAR" || t.CommissionStatus == "DIPOTONG" {
				paidCommission += t.CommissionAmount
			}
			if t.CampusStatus != "DISETOR" {
				bill := t.Amount
				if mode == "potong_setoran" {
					bill -= t.CommissionAmount
				}
				unreturnedBills += bill
			}
		}
	}

	var deposited float64
	var settlementPending float64
	for _, s := range settlements {
		if s.Status == "VERIFIED" {
			deposited += s.Amount
		} else if s.Status == "SUBMITTED" {
			settlementPending += s.Amount
		}
	}

	return &dto.ReferralSummaryResponse{
		TotalUsers:               totalUsers,
		AllTransactionsSuccess:   allSuccess,
		MyBillToCampus:           unreturnedBills,
		DepositedToCampus:        deposited,
		PendingVerificationQueue: pendingQueue,
		MyCommissionTotal:        totalCommission,
		MyCommissionPaid:         paidCommission,
		MyCommissionPending:      totalCommission - paidCommission,
		SettlementPendingAmount:  settlementPending,
		UpdatedAt:                time.Now(),
	}, nil
}

func (s *referralService) ListCodeUsers(ctx context.Context, codeID, userID uuid.UUID, isAdmin bool, q, approveStatus, campusStatus string) ([]dto.ReferralUserItem, error) {
	c, err := s.repo.GetCodeByID(ctx, codeID)
	if err != nil || c == nil {
		return nil, ErrReferralNotFound
	}
	if !isAdmin {
		isOwner := false
		for _, pic := range c.PICs {
			if pic.UserID == userID {
				isOwner = true
				break
			}
		}
		if !isOwner {
			return nil, ErrReferralNotFound
		}
	}

	txs, _ := s.repo.ListTransactionsByCode(ctx, codeID, approveStatus, campusStatus)
	usages, _ := s.repo.ListUsagesByCode(ctx, codeID)

	items := make([]dto.ReferralUserItem, 0)
	no := 1
	for _, u := range usages {
		var matchedTx *model.ReferralTransaction
		for _, t := range txs {
			if t.UsageID == u.ID {
				matchedTx = &t
				break
			}
		}

		// Ambil data camaba dari pmb_auth dan pmb_app
		var userData struct {
			FullName string
			Email    string
		}
		_ = s.baseDB.Table("pmb_auth.users").Select("full_name, email").Where("id = ?", u.UserID).Scan(&userData).Error

		var biodata struct {
			NIK  string
			NISN string
		}
		_ = s.baseDB.Table("identity.student_biodata").Select("nik, nisn").Where("user_id = ?", u.UserID).Scan(&biodata).Error

		nikDisplay := MaskNIK(biodata.NIK)
		if isAdmin {
			nikDisplay = biodata.NIK
		}

		item := dto.ReferralUserItem{
			No:                 no,
			ID:                 u.ID,
			ApplicationID:      u.ApplicationID,
			Name:               userData.FullName,
			Email:              userData.Email,
			NIK:                nikDisplay,
			NISN:               biodata.NISN,
			PathName:           "Reguler",
			Type:               "REGISTRATION",
			ApproveStatus:      "GRATIS",
			ApproveStatusLabel: "Gratis",
			CampusStatus:       "-",
			CampusStatusLabel:  "-",
			AttachedAt:         u.AttachedAt,
			ProgramName:        "-",
			RegistrationStep:   "Biodata",
		}

		if matchedTx != nil {
			item.Amount = matchedTx.Amount
			item.ApproveStatus = matchedTx.ApproveStatus
			item.ApproveStatusLabel = matchedTx.ApproveStatus
			item.CampusStatus = matchedTx.CampusStatus
			item.CampusStatusLabel = matchedTx.CampusStatus
			item.CommissionAmount = matchedTx.CommissionAmount
		}

		if q != "" {
			lq := strings.ToLower(q)
			if !strings.Contains(strings.ToLower(item.Name), lq) && !strings.Contains(strings.ToLower(item.Email), lq) {
				continue
			}
		}

		items = append(items, item)
		no++
	}

	return items, nil
}

func (s *referralService) ApproveTransactions(ctx context.Context, txIDs []uuid.UUID, actorID uuid.UUID, isAdmin bool) error {
	now := time.Now()
	for _, id := range txIDs {
		tx, err := s.repo.GetTransactionByID(ctx, id)
		if err != nil || tx == nil || tx.ApproveStatus != "PENDING" {
			continue
		}

		code, _ := s.repo.GetCodeByID(ctx, tx.CodeID)
		if code == nil {
			continue
		}

		// Hitung komisi snapshot
		var commType string = code.CommissionType
		var commVal float64 = code.CommissionValue
		var commAmt float64 = 0
		var commStatus string = "BELUM_DIBAYAR"

		// Cek apakah jalur mendukung komisi
		var path struct {
			ReferralCommissionEnabled bool
			RegistrationFee           float64
		}
		_ = s.baseDB.Table("admission.admission_paths").
			Select("referral_commission_enabled, registration_fee").
			Joins("JOIN admission.applications ON applications.path_id = admission_paths.id").
			Where("applications.id = (SELECT application_id FROM referral.referral_usages WHERE id = ?)", tx.UsageID).
			Scan(&path).Error

		if path.RegistrationFee == 0 || !path.ReferralCommissionEnabled || commType == "tidak_ada" {
			commStatus = "TIDAK_ADA"
			commAmt = 0
		} else {
			if commType == "persen" {
				commAmt = math.Round((commVal / 100.0) * tx.Amount)
			} else if commType == "nominal" {
				commAmt = commVal
				if commAmt > tx.Amount {
					commAmt = tx.Amount
				}
			}
		}

		tx.ApproveStatus = "SUCCESS"
		tx.CampusStatus = "TAGIHAN"
		tx.ApprovedBy = &actorID
		tx.ApprovedAt = &now
		tx.CommissionTypeSnapshot = commType
		tx.CommissionValueSnapshot = commVal
		tx.CommissionAmount = commAmt
		tx.CommissionStatus = commStatus

		_ = s.repo.UpdateTransaction(ctx, tx)

		// Kunci usage
		if usage, _ := s.repo.GetUsageByID(ctx, tx.UsageID); usage != nil {
			usage.LockedAt = &now
			_ = s.repo.UpdateUsage(ctx, usage)
		}

		// Tandai tagihan formulir lunas di modul finance
		_ = s.baseDB.Table("finance.invoices").
			Where("id = ?", tx.InvoiceID).
			Updates(map[string]interface{}{
				"status":     "paid",
				"updated_at": now,
				"notes":      fmt.Sprintf("Dibayar melalui guru BK — %s", code.InstitutionName),
			}).Error
	}
	return nil
}

func (s *referralService) RejectTransaction(ctx context.Context, txID uuid.UUID, reason string, actorID uuid.UUID, isAdmin bool) error {
	tx, err := s.repo.GetTransactionByID(ctx, txID)
	if err != nil || tx == nil {
		return errors.New("transaksi tidak ditemukan")
	}

	tx.ApproveStatus = "REJECTED"
	tx.RejectReason = reason
	tx.ApprovedBy = &actorID
	now := time.Now()
	tx.ApprovedAt = &now

	_ = s.repo.UpdateTransaction(ctx, tx)

	// Kembalikan status invoice agar bisa dibayar mandiri
	_ = s.baseDB.Table("finance.invoices").
		Where("id = ?", tx.InvoiceID).
		Update("notes", "REJECTED_BY_PARTNER:"+reason).Error

	return nil
}

func (s *referralService) SubmitSettlement(ctx context.Context, codeID, userID uuid.UUID, req dto.SubmitSettlementRequest) (*model.ReferralSettlement, error) {
	c, err := s.repo.GetCodeByID(ctx, codeID)
	if err != nil || c == nil {
		return nil, ErrReferralNotFound
	}

	txs, err := s.repo.ListTransactionsByCode(ctx, codeID, "SUCCESS", "TAGIHAN")
	if err != nil || len(txs) == 0 {
		return nil, errors.New("tidak ada transaksi tagihan yang dapat disetorkan")
	}

	mode := c.CommissionSettlementMode
	if mode == "" {
		st, _ := s.repo.GetSettings(ctx)
		if st != nil {
			mode = st.CommissionSettlementMode
		}
	}

	var gross float64
	var commDeducted float64
	for _, t := range txs {
		gross += t.Amount
		if mode == "potong_setoran" {
			commDeducted += t.CommissionAmount
		}
	}

	tDate, _ := time.Parse("2006-01-02", req.TransferDate)
	if tDate.IsZero() {
		tDate = time.Now()
	}

	now := time.Now()
	settlement := model.ReferralSettlement{
		ID:                 uuid.New(),
		CodeID:             codeID,
		GrossAmount:        gross,
		CommissionDeducted: commDeducted,
		Amount:             gross - commDeducted,
		TransferDate:       tDate,
		BankName:           req.BankName,
		ReferenceNo:        req.ReferenceNo,
		ProofObjectKey:     req.ProofObjectKey,
		Notes:              req.Notes,
		Status:             "SUBMITTED",
		SubmittedBy:        userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.CreateSettlement(ctx, &settlement); err != nil {
		return nil, err
	}

	// Update campus_status transaksi
	for _, t := range txs {
		t.CampusStatus = "MENUNGGU_VERIFIKASI"
		t.SettlementID = &settlement.ID
		_ = s.repo.UpdateTransaction(ctx, &t)
	}

	return &settlement, nil
}

func (s *referralService) ListSettlements(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralSettlement, error) {
	return s.repo.ListSettlements(ctx, codeID, status)
}

func (s *referralService) VerifySettlement(ctx context.Context, settlementID, actorID uuid.UUID) error {
	st, err := s.repo.GetSettlementByID(ctx, settlementID)
	if err != nil || st == nil {
		return errors.New("setoran tidak ditemukan")
	}

	now := time.Now()
	st.Status = "VERIFIED"
	st.VerifiedBy = &actorID
	st.VerifiedAt = &now
	_ = s.repo.UpdateSettlement(ctx, st)

	// Update transaksi terkait
	txs, _ := s.repo.ListTransactionsByCode(ctx, st.CodeID, "SUCCESS", "MENUNGGU_VERIFIKASI")
	for _, t := range txs {
		if t.SettlementID != nil && *t.SettlementID == st.ID {
			t.CampusStatus = "DISETOR"
			if st.CommissionDeducted > 0 && t.CommissionAmount > 0 {
				t.CommissionStatus = "DIPOTONG"
			}
			_ = s.repo.UpdateTransaction(ctx, &t)
		}
	}

	return nil
}

func (s *referralService) RejectSettlement(ctx context.Context, settlementID uuid.UUID, reason string, actorID uuid.UUID) error {
	st, err := s.repo.GetSettlementByID(ctx, settlementID)
	if err != nil || st == nil {
		return errors.New("setoran tidak ditemukan")
	}

	now := time.Now()
	st.Status = "REJECTED"
	st.RejectReason = reason
	st.VerifiedBy = &actorID
	st.VerifiedAt = &now
	_ = s.repo.UpdateSettlement(ctx, st)

	// Kembalikan transaksi ke status TAGIHAN
	txs, _ := s.repo.ListTransactionsByCode(ctx, st.CodeID, "SUCCESS", "MENUNGGU_VERIFIKASI")
	for _, t := range txs {
		if t.SettlementID != nil && *t.SettlementID == st.ID {
			t.CampusStatus = "TAGIHAN"
			t.SettlementID = nil
			_ = s.repo.UpdateTransaction(ctx, &t)
		}
	}

	return nil
}

func (s *referralService) ListCommissions(ctx context.Context, codeID *uuid.UUID, status string) ([]model.ReferralTransaction, error) {
	if codeID == nil {
		return []model.ReferralTransaction{}, nil
	}
	return s.repo.ListTransactionsByCode(ctx, *codeID, "SUCCESS", "")
}

func (s *referralService) CreatePayout(ctx context.Context, req dto.CreateCommissionPayoutRequest, actorID uuid.UUID) error {
	now := time.Now()
	payout := model.ReferralCommissionPayout{
		ID:             uuid.New(),
		CodeID:         req.CodeID,
		Amount:         req.Amount,
		PaidAt:         now,
		Method:         req.Method,
		ReferenceNo:    req.ReferenceNo,
		ProofObjectKey: req.ProofObjectKey,
		PaidBy:         actorID,
		CreatedAt:      now,
	}

	if err := s.repo.CreatePayout(ctx, &payout); err != nil {
		return err
	}

	for _, txID := range req.TransactionIDs {
		tx, _ := s.repo.GetTransactionByID(ctx, txID)
		if tx != nil {
			tx.CommissionStatus = "DIBAYAR"
			tx.CommissionPayoutID = &payout.ID
			_ = s.repo.UpdateTransaction(ctx, tx)
		}
	}
	return nil
}

func (s *referralService) GetPartnerDashboard(ctx context.Context, userID uuid.UUID) (*dto.PartnerDashboardResponse, error) {
	codes, _ := s.repo.ListCodesByPIC(ctx, userID)

	var user struct {
		FullName string
		Email    string
		Phone    string
	}
	_ = s.baseDB.Table("pmb_auth.users").Select("full_name, email, phone").Where("id = ?", userID).Scan(&user).Error

	var combinedSummary dto.ReferralSummaryResponse
	combinedSummary.UpdatedAt = time.Now()

	pendingItems := make([]dto.ReferralUserItem, 0)
	for _, c := range codes {
		sm, err := s.calculateSummary(ctx, c.ID, &c)
		if err == nil {
			combinedSummary.TotalUsers += sm.TotalUsers
			combinedSummary.AllTransactionsSuccess += sm.AllTransactionsSuccess
			combinedSummary.MyBillToCampus += sm.MyBillToCampus
			combinedSummary.DepositedToCampus += sm.DepositedToCampus
			combinedSummary.PendingVerificationQueue += sm.PendingVerificationQueue
			combinedSummary.MyCommissionTotal += sm.MyCommissionTotal
			combinedSummary.MyCommissionPaid += sm.MyCommissionPaid
			combinedSummary.MyCommissionPending += sm.MyCommissionPending
			combinedSummary.SettlementPendingAmount += sm.SettlementPendingAmount
		}
		uItems, _ := s.ListCodeUsers(ctx, c.ID, userID, false, "", "PENDING", "")
		if len(pendingItems) < 5 {
			for _, item := range uItems {
				pendingItems = append(pendingItems, item)
				if len(pendingItems) >= 5 {
					break
				}
			}
		}
	}

	var instProfile struct {
		Phone             string
		WAGroupGuruBKURL  string
		WAGroupGuruBKName string
	}
	_ = s.baseDB.Table("organization.institution_profiles").
		Select("phone, wa_group_guru_bk_url, wa_group_guru_bk_name").
		Limit(1).Scan(&instProfile).Error

	resp := &dto.PartnerDashboardResponse{
		TotalCodes:          len(codes),
		Summary:             combinedSummary,
		PendingTransactions: pendingItems,
		WAGroupGuruBKURL:    instProfile.WAGroupGuruBKURL,
		WAGroupGuruBKName:   instProfile.WAGroupGuruBKName,
		CampusContact:       instProfile.Phone,
	}
	resp.Profile.ID = userID
	resp.Profile.FullName = user.FullName
	resp.Profile.Email = user.Email
	resp.Profile.Phone = user.Phone

	return resp, nil
}

func (s *referralService) ListAdminCodes(ctx context.Context, q, status string) ([]model.ReferralCode, error) {
	return s.repo.ListCodes(ctx, q, status)
}

func (s *referralService) CreateCode(ctx context.Context, code *model.ReferralCode) error {
	return s.repo.CreateCode(ctx, code)
}

func (s *referralService) UpdateCode(ctx context.Context, code *model.ReferralCode) error {
	return s.repo.UpdateCode(ctx, code)
}

func (s *referralService) ImportPICsCSV(ctx context.Context, r io.Reader, actorID uuid.UUID) (created int, updated int, failed int, errs []string, err error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return 0, 0, 0, nil, err
	}

	if len(rows) <= 1 {
		return 0, 0, 0, nil, nil
	}

	// Format header: nama, email, no_hp, nama_institusi, jenis_institusi, npsn, kode, tipe_komisi, nilai_komisi, kirim_aktivasi
	for i, row := range rows[1:] {
		rowNum := i + 2
		if len(row) < 4 {
			failed++
			errs = append(errs, fmt.Sprintf("Baris %d: kolom tidak lengkap", rowNum))
			continue
		}

		name := strings.TrimSpace(row[0])
		email := strings.ToLower(strings.TrimSpace(row[1]))
		phone := strings.TrimSpace(row[2])
		instName := strings.TrimSpace(row[3])

		if name == "" || email == "" || instName == "" {
			failed++
			errs = append(errs, fmt.Sprintf("Baris %d: nama, email, atau nama institusi kosong", rowNum))
			continue
		}

		instType := "sekolah"
		if len(row) > 4 && strings.TrimSpace(row[4]) != "" {
			instType = strings.TrimSpace(row[4])
		}
		npsn := ""
		if len(row) > 5 {
			npsn = strings.TrimSpace(row[5])
		}
		codeVal := ""
		if len(row) > 6 && strings.TrimSpace(row[6]) != "" {
			codeVal = strings.ToUpper(strings.TrimSpace(row[6]))
		} else {
			// Auto generate: 6 char institusi + 2 angka
			cleanInst := strings.ToUpper(regexpAlphanum(instName))
			if len(cleanInst) > 6 {
				cleanInst = cleanInst[:6]
			}
			codeVal = fmt.Sprintf("%s%02d", cleanInst, rand.Intn(90)+10)
		}

		commType := "persen"
		if len(row) > 7 && strings.TrimSpace(row[7]) != "" {
			commType = strings.TrimSpace(row[7])
		}
		var commVal float64 = 10
		if len(row) > 8 && strings.TrimSpace(row[8]) != "" {
			val, _ := strconv.ParseFloat(strings.TrimSpace(row[8]), 64)
			if val >= 0 {
				commVal = val
			}
		}

		// Cari atau buat user di pmb_auth
		var existingUser struct {
			ID uuid.UUID
		}
		_ = s.baseDB.Table("pmb_auth.users").Select("id").Where("email = ?", email).Scan(&existingUser).Error

		userID := existingUser.ID
		if userID == uuid.Nil {
			userID = uuid.New()
			now := time.Now()
			_ = s.baseDB.Exec(`
				INSERT INTO pmb_auth.users (id, email, full_name, phone, status, password_hash, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'active', '$2a$10$abcdefghijklmnopqrstuv', ?, ?)
			`, userID, email, name, phone, now, now).Error

			// Hubungkan ke role guru_bk dan service account pmb-mitra
			var roleGuruBK struct {
				ID uint
			}
			_ = s.baseDB.Table("pmb_auth.roles").Select("id").Where("code = 'guru_bk'").Scan(&roleGuruBK).Error

			var saMitra struct {
				ID uuid.UUID
			}
			_ = s.baseDB.Table("pmb_auth.service_accounts").Select("id").Where("code = 'pmb-mitra'").Scan(&saMitra).Error

			if saMitra.ID != uuid.Nil {
				uaID := uuid.New()
				_ = s.baseDB.Exec(`
					INSERT INTO pmb_auth.user_accounts (id, user_id, service_account_id, status, granted_at, created_at, updated_at)
					VALUES (?, ?, ?, 'active', ?, ?, ?)
				`, uaID, userID, saMitra.ID, now, now, now).Error

				if roleGuruBK.ID > 0 {
					_ = s.baseDB.Exec(`
						INSERT INTO pmb_auth.user_account_roles (user_account_id, role_id, scope_type, scope_id, created_at)
						VALUES (?, ?, '', '', ?)
					`, uaID, roleGuruBK.ID, now).Error
				}
			}
		}

		// Buat atau update ReferralCode
		existingCode, _ := s.repo.GetCodeByCode(ctx, codeVal)
		now := time.Now()
		if existingCode != nil {
			existingCode.InstitutionName = instName
			existingCode.InstitutionType = instType
			existingCode.NPSN = npsn
			existingCode.CommissionType = commType
			existingCode.CommissionValue = commVal
			_ = s.repo.UpdateCode(ctx, existingCode)
			updated++
		} else {
			newCode := model.ReferralCode{
				ID:              uuid.New(),
				Code:            codeVal,
				PaymentCode:     codeVal,
				InstitutionName: instName,
				InstitutionType: instType,
				NPSN:            npsn,
				Status:          "active",
				CommissionType:  commType,
				CommissionValue: commVal,
				CreatedBy:       &actorID,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			_ = s.repo.CreateCode(ctx, &newCode)
			_ = s.repo.AddPIC(ctx, &model.ReferralCodePIC{
				CodeID:    newCode.ID,
				UserID:    userID,
				IsPrimary: true,
				CreatedAt: now,
			})
			created++
		}
	}

	return created, updated, failed, errs, nil
}

func regexpAlphanum(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *referralService) GetSettings(ctx context.Context) (*model.ReferralSetting, error) {
	return s.repo.GetSettings(ctx)
}

func (s *referralService) UpdateSettings(ctx context.Context, st *model.ReferralSetting) error {
	return s.repo.UpdateSettings(ctx, st)
}

func (s *referralService) ChangeApplicationReferral(ctx context.Context, appID, newCodeID, actorID uuid.UUID, reason string) error {
	u, err := s.repo.GetUsageByApplication(ctx, appID)
	if err != nil || u == nil {
		return errors.New("penggunaan referal tidak ditemukan")
	}

	c, err := s.repo.GetCodeByID(ctx, newCodeID)
	if err != nil || c == nil {
		return ErrReferralNotFound
	}

	u.CodeID = c.ID
	u.ChangedBy = &actorID
	u.ChangeReason = reason
	return s.repo.UpdateUsage(ctx, u)
}
