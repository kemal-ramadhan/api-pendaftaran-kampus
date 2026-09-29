package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/nim-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrFormatNotFound    = errors.New("format NIM tidak ditemukan")
	ErrIssuanceNotFound  = errors.New("data penerbitan NIM tidak ditemukan")
	ErrInvalidParameter = errors.New("parameter penerbitan NIM tidak valid")
)

type NIMService interface {
	IssueNIM(ctx context.Context, req *model.IssueNIMRequest) (*model.IssueNIMResponse, error)
	GetIssuanceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error)
	ListIssuances(ctx context.Context, filter model.IssuanceFilter) ([]model.NIMIssuance, int64, error)
	ListFormats(ctx context.Context) ([]model.NIMFormat, error)
	GetFormatByID(ctx context.Context, id uuid.UUID) (*model.NIMFormat, error)
	CreateFormat(ctx context.Context, req *model.CreateNIMFormatRequest) (*model.NIMFormat, error)
	UpdateFormat(ctx context.Context, id uuid.UUID, req *model.UpdateNIMFormatRequest) (*model.NIMFormat, error)
	VoidNIM(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error)
}

type nimService struct {
	cfg     *config.Config
	repo    repository.NIMRepository
	studCli internalclient.Client
	authCli internalclient.Client
	admsCli internalclient.Client
	notfCli internalclient.Client
	orgCli  internalclient.Client
}

func NewNIMService(
	cfg *config.Config,
	repo repository.NIMRepository,
	studCli internalclient.Client,
	authCli internalclient.Client,
	admsCli internalclient.Client,
	notfCli internalclient.Client,
	orgCli internalclient.Client,
) NIMService {
	return &nimService{
		cfg:     cfg,
		repo:    repo,
		studCli: studCli,
		authCli: authCli,
		admsCli: admsCli,
		notfCli: notfCli,
		orgCli:  orgCli,
	}
}

func (s *nimService) IssueNIM(ctx context.Context, req *model.IssueNIMRequest) (*model.IssueNIMResponse, error) {
	if req.ApplicationID == uuid.Nil || req.UserID == uuid.Nil || req.StudyProgramCode == "" || req.EntryYear <= 0 {
		return nil, ErrInvalidParameter
	}

	// 1. Idempotensi: Cek apakah application_id sudah pernah diterbitkan NIM-nya
	existing, err := s.repo.GetIssuanceByApplicationID(ctx, req.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa riwayat penerbitan NIM: %w", err)
	}
	if existing != nil {
		return &model.IssueNIMResponse{
			NIM:              existing.NIM,
			ApplicationID:    existing.ApplicationID,
			UserID:           existing.UserID,
			StudyProgramCode: existing.StudyProgramCode,
			EntryYear:        existing.EntryYear,
			SelectionPath:    existing.SelectionPath,
			SequenceNumber:   existing.SequenceNumber,
			IssuedAt:         existing.IssuedAt,
			IsIdempotent:     true,
		}, nil
	}

	// 2. Transaksi Database dengan SELECT ... FOR UPDATE untuk mengunci sequence counter
	var tx *gorm.DB
	if s.repo.DB() != nil && s.repo.DB().Config != nil {
		tx = s.repo.DB().Begin()
		if tx.Error != nil {
			return nil, tx.Error
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
	}

	// Ambil format template aktif
	fmtModel, errFmt := s.repo.GetFormatForProgram(ctx, req.StudyProgramCode)
	pattern := "{YY}{nim_code_prodi}{jalur_digit}{urut:4}"
	if errFmt == nil && fmtModel != nil && fmtModel.Pattern != "" {
		pattern = fmtModel.Pattern
	}

	// Dapatkan nomor urut berikutnya dengan lock FOR UPDATE
	seqNumber, errSeq := s.repo.GetNextSequenceWithLock(ctx, tx, req.EntryYear, req.StudyProgramCode)
	if errSeq != nil {
		if tx != nil {
			tx.Rollback()
		}
		return nil, fmt.Errorf("gagal mendapatkan nomor urut sequence NIM: %w", errSeq)
	}

	// Dapatkan nim_code dari prodi
	nimCodeProdi := s.getProdiNimCode(ctx, req.StudyProgramCode)
	pathDigit := s.getPathDigit(req.SelectionPath)

	generatedNIM := renderNIM(pattern, req.EntryYear, nimCodeProdi, pathDigit, seqNumber)

	now := time.Now()
	issuance := &model.NIMIssuance{
		ID:               uuid.New(),
		ApplicationID:    req.ApplicationID,
		UserID:           req.UserID,
		NIM:              generatedNIM,
		EntryYear:        req.EntryYear,
		StudyProgramCode: req.StudyProgramCode,
		SelectionPath:    req.SelectionPath,
		SequenceNumber:   seqNumber,
		IssuedAt:         now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if errCreate := s.repo.CreateIssuanceInTx(ctx, tx, issuance); errCreate != nil {
		if tx != nil {
			tx.Rollback()
		}
		return nil, fmt.Errorf("gagal menyimpan penerbitan NIM: %w", errCreate)
	}

	if tx != nil {
		if errCommit := tx.Commit().Error; errCommit != nil {
			return nil, fmt.Errorf("gagal commit transaksi penerbitan NIM: %w", errCommit)
		}
	}

	// 3. Orkestrasi Inter-Service Post-Commit (Non-blocking & Graceful Failure Handling)
	go s.orchestrateDownstreamServices(context.Background(), issuance)

	return &model.IssueNIMResponse{
		NIM:              issuance.NIM,
		ApplicationID:    issuance.ApplicationID,
		UserID:           issuance.UserID,
		StudyProgramCode: issuance.StudyProgramCode,
		EntryYear:        issuance.EntryYear,
		SelectionPath:    issuance.SelectionPath,
		SequenceNumber:   issuance.SequenceNumber,
		IssuedAt:         issuance.IssuedAt,
		IsIdempotent:     false,
	}, nil
}

func (s *nimService) GetIssuanceByApplicationID(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	issuance, err := s.repo.GetIssuanceByApplicationID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if issuance == nil {
		return nil, ErrIssuanceNotFound
	}
	return issuance, nil
}

func (s *nimService) ListIssuances(ctx context.Context, filter model.IssuanceFilter) ([]model.NIMIssuance, int64, error) {
	return s.repo.ListIssuances(ctx, filter)
}

func (s *nimService) ListFormats(ctx context.Context) ([]model.NIMFormat, error) {
	return s.repo.ListFormats(ctx)
}

func (s *nimService) GetFormatByID(ctx context.Context, id uuid.UUID) (*model.NIMFormat, error) {
	fmtItem, err := s.repo.GetFormatByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if fmtItem == nil {
		return nil, ErrFormatNotFound
	}
	return fmtItem, nil
}

func (s *nimService) CreateFormat(ctx context.Context, req *model.CreateNIMFormatRequest) (*model.NIMFormat, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	newFmt := &model.NIMFormat{
		ID:               uuid.New(),
		StudyProgramCode: req.StudyProgramCode,
		Pattern:          strings.TrimSpace(req.Pattern),
		Description:      strings.TrimSpace(req.Description),
		IsActive:         isActive,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.repo.CreateFormat(ctx, newFmt); err != nil {
		return nil, err
	}

	return newFmt, nil
}

func (s *nimService) UpdateFormat(ctx context.Context, id uuid.UUID, req *model.UpdateNIMFormatRequest) (*model.NIMFormat, error) {
	fmtItem, err := s.repo.GetFormatByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if fmtItem == nil {
		return nil, ErrFormatNotFound
	}

	if req.Pattern != "" {
		fmtItem.Pattern = strings.TrimSpace(req.Pattern)
	}
	if req.Description != "" {
		fmtItem.Description = strings.TrimSpace(req.Description)
	}
	if req.IsActive != nil {
		fmtItem.IsActive = *req.IsActive
	}
	fmtItem.UpdatedAt = time.Now()

	if err := s.repo.UpdateFormat(ctx, fmtItem); err != nil {
		return nil, err
	}

	return fmtItem, nil
}

// -----------------------------------------------------------------------------
// Helper Methods & Inter-Service Downstream Orchestration
// -----------------------------------------------------------------------------

func (s *nimService) orchestrateDownstreamServices(ctx context.Context, issuance *model.NIMIssuance) {
	// 1. Update Student Service: Simpan NIM ke tabel students & ubah status ke active
	if s.studCli != nil {
		path := fmt.Sprintf("/internal/students/by-application/%s/nim", issuance.ApplicationID.String())
		payload := map[string]interface{}{
			"nim": issuance.NIM,
		}
		_ = s.studCli.Put(ctx, path, payload, nil)
	}

	// 2. Update Auth Service: Tambah role 'mahasiswa' dan cabut sesi aktif
	if s.authCli != nil {
		path := fmt.Sprintf("/internal/users/%s/roles", issuance.UserID.String())
		payload := map[string]interface{}{
			"role": "mahasiswa",
		}
		_ = s.authCli.Post(ctx, path, payload, nil)
	}

	// 3. Update Admission Service: Mutasi status pendaftaran menjadi 'enrolled'
	if s.admsCli != nil {
		path := fmt.Sprintf("/internal/applications/%s/status", issuance.ApplicationID.String())
		payload := map[string]interface{}{
			"status": "enrolled",
			"notes":  fmt.Sprintf("NIM %s berhasil diterbitkan", issuance.NIM),
		}
		_ = s.admsCli.Put(ctx, path, payload, nil)
	}

	// 4. Kirim Notifikasi: Antrekan notifikasi 'nim_issued'
	if s.notfCli != nil {
		payload := map[string]interface{}{
			"idempotency_key": fmt.Sprintf("nim_issued_%s", issuance.ApplicationID.String()),
			"user_id":         issuance.UserID,
			"channel":         "email",
			"template":        "nim_issued",
			"data": map[string]interface{}{
				"nim":                issuance.NIM,
				"study_program_code": issuance.StudyProgramCode,
				"entry_year":         issuance.EntryYear,
			},
		}
		_ = s.notfCli.Post(ctx, "/internal/notifications", payload, nil)
	}
}

func (s *nimService) getProdiNimCode(ctx context.Context, prodiCode string) string {
	// Mapping standar default jika organization service tidak merespons
	defaults := map[string]string{
		"TI": "55",
		"SI": "57",
		"MI": "58",
		"TK": "59",
		"AK": "61",
		"MN": "62",
	}

	normCode := strings.ToUpper(strings.TrimSpace(prodiCode))
	if code, ok := defaults[normCode]; ok {
		return code
	}

	// Coba ambil dari Organization Service jika ada
	if s.orgCli != nil {
		type OrgProdiResp struct {
			NIMCode string `json:"nim_code"`
		}
		var resp OrgProdiResp
		if err := s.orgCli.Get(ctx, fmt.Sprintf("/internal/study-programs/%s", normCode), nil, &resp); err == nil && resp.NIMCode != "" {
			return resp.NIMCode
		}
	}

	return "10"
}

func (s *nimService) getPathDigit(selectionPath string) string {
	lower := strings.ToLower(strings.TrimSpace(selectionPath))
	switch lower {
	case "reguler", "regular":
		return "1"
	case "prestasi":
		return "2"
	case "beasiswa":
		return "3"
	case "pindahan":
		return "4"
	default:
		return "1"
	}
}

func renderNIM(pattern string, year int, prodiNimCode, pathDigit string, seqNumber int) string {
	if pattern == "" {
		pattern = "{YY}{nim_code_prodi}{jalur_digit}{urut:4}"
	}
	yy := fmt.Sprintf("%02d", year%100)
	urutStr := fmt.Sprintf("%04d", seqNumber)

	res := pattern
	res = strings.ReplaceAll(res, "{YY}", yy)
	res = strings.ReplaceAll(res, "{nim_code_prodi}", prodiNimCode)
	res = strings.ReplaceAll(res, "{jalur_digit}", pathDigit)
	res = strings.ReplaceAll(res, "{urut:4}", urutStr)
	res = strings.ReplaceAll(res, "{urut}", urutStr)
	return res
}

func (s *nimService) VoidNIM(ctx context.Context, appID uuid.UUID) (*model.NIMIssuance, error) {
	return s.repo.VoidIssuance(ctx, appID)
}

