package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/repository"
)

var (
	ErrScanRateLimitExceeded = errors.New("batas pemindaian tercapai (maksimal 10 kali per jam)")
	ErrScanFileTooLarge      = errors.New("ukuran file KTP melebihi batas 5 MB")
	ErrScanInvalidFileType   = errors.New("format file KTP harus berupa JPG, PNG, atau PDF")
	ErrOCRUpstreamFailed     = errors.New("gagal menghubungi layanan OCR internal")
	ErrScanAlreadyConfirmed  = errors.New("pemindaian KTP ini sudah dikonfirmasi sebelumnya")
)

type KTPScanService interface {
	InitiateKTPScan(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, filename, contentType string) (*dto.KTPScanInitiateResponse, error)
	GetKTPScanStatus(ctx context.Context, scanID, userID uuid.UUID) (*dto.KTPScanStatusResponse, error)
	ConfirmKTPScan(ctx context.Context, scanID, userID uuid.UUID, req dto.KTPConfirmRequest) (*dto.ProfileResponse, error)
}

type ktpScanService struct {
	ktpScanRepo   repository.KTPScanRepository
	identitySvc   IdentityService
	storageClient storage.Client
	ocrClient     internalclient.Client
	ktpBucket     string
}

func NewKTPScanService(
	ktpScanRepo repository.KTPScanRepository,
	identitySvc IdentityService,
	storageClient storage.Client,
	ocrClient internalclient.Client,
	ktpBucket string,
) KTPScanService {
	if ktpBucket == "" {
		ktpBucket = "pmb-identity-ktp"
	}
	return &ktpScanService{
		ktpScanRepo:   ktpScanRepo,
		identitySvc:   identitySvc,
		storageClient: storageClient,
		ocrClient:     ocrClient,
		ktpBucket:     ktpBucket,
	}
}

func (s *ktpScanService) InitiateKTPScan(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, filename, contentType string) (*dto.KTPScanInitiateResponse, error) {
	// 1. Rate limiting: maks 10 scan per user per jam
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	count, err := s.ktpScanRepo.CountScansSince(ctx, userID, oneHourAgo)
	if err == nil && count >= 10 {
		return nil, ErrScanRateLimitExceeded
	}

	// 2. Validasi ukuran (maks 5MB)
	if size > 5*1024*1024 {
		return nil, ErrScanFileTooLarge
	}

	// 3. Baca seluruh bytes file agar bisa di-upload ke MinIO dan dikirim ke OCR
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca data file gambar: %w", err)
	}

	allowedTypes := []string{"image/jpeg", "image/png", "application/pdf"}
	var validatedReader io.Reader = bytes.NewReader(fileBytes)
	var sniffedType = contentType

	if s.storageClient != nil {
		vr, st, err := s.storageClient.SniffAndValidate(bytes.NewReader(fileBytes), allowedTypes)
		if err != nil {
			return nil, ErrScanInvalidFileType
		}
		validatedReader = vr
		sniffedType = st
		_ = s.storageClient.EnsureBucket(ctx, s.ktpBucket)
	}

	scanID := uuid.New()
	ext := "jpg"
	if strings.Contains(sniffedType, "png") {
		ext = "png"
	} else if strings.Contains(sniffedType, "pdf") {
		ext = "pdf"
	}

	objectKey := fmt.Sprintf("ktp/%s/%s.%s", userID.String(), scanID.String(), ext)

	// 4. Simpan gambar asli ke MinIO bucket privat
	if s.storageClient != nil {
		err = s.storageClient.Upload(ctx, s.ktpBucket, objectKey, validatedReader, int64(len(fileBytes)), sniffedType)
		if err != nil {
			return nil, fmt.Errorf("gagal mengunggah gambar KTP ke penyimpanan: %w", err)
		}
	}

	// 5. Kirim ke OCR Service via Internal HTTP Multipart
	var ocrResp struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}

	if s.ocrClient != nil {
		err = s.ocrClient.PostMultipart(
			ctx,
			"/internal/ocr/ktp",
			"file",
			filename,
			bytes.NewReader(fileBytes),
			nil,
			&ocrResp,
		)
		if err != nil {
			// Simpan status failed bila gagal kirim ke OCR
			_ = s.ktpScanRepo.Create(ctx, &model.KTPScan{
				ID:                 scanID,
				UserID:             userID,
				ObjectKey:          objectKey,
				Status:             "failed",
				Reason:             "OCR_SERVICE_UNAVAILABLE",
				FieldConfidences:   "{}",
				FieldsEditedByUser: "[]",
			})
			return nil, fmt.Errorf("%w: %v", ErrOCRUpstreamFailed, err)
		}
	} else {
		// Mock job ID jika tanpa OCR service aktif
		ocrResp.JobID = "job-" + scanID.String()
		ocrResp.Status = "completed"
	}

	scanRecord := &model.KTPScan{
		ID:                 scanID,
		UserID:             userID,
		ObjectKey:          objectKey,
		JobID:              ocrResp.JobID,
		Status:             ocrResp.Status,
		FieldConfidences:   "{}",
		FieldsEditedByUser: "[]",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.ktpScanRepo.Create(ctx, scanRecord); err != nil {
		return nil, err
	}

	return &dto.KTPScanInitiateResponse{
		ScanID: scanID.String(),
		Status: ocrResp.Status,
	}, nil
}

func (s *ktpScanService) GetKTPScanStatus(ctx context.Context, scanID, userID uuid.UUID) (*dto.KTPScanStatusResponse, error) {
	scan, err := s.ktpScanRepo.FindByID(ctx, scanID)
	if err != nil {
		return nil, err
	}

	if scan.UserID != userID {
		return nil, ErrUnauthorizedAccess
	}

	// Jika masih processing dan punya job_id di ocr service, poll status terbaru
	if scan.Status == "processing" && scan.JobID != "" && s.ocrClient != nil {
		var jobDetail struct {
			JobID         string                   `json:"job_id"`
			Status        string                   `json:"status"`
			QualityScore  float64                  `json:"quality_score"`
			Enhanced      bool                     `json:"enhanced"`
			Reason        string                   `json:"reason"`
			ReasonMessage string                   `json:"reason_message"`
			Fields        map[string]*dto.KTPFieldItem `json:"fields"`
		}

		err := s.ocrClient.Get(ctx, "/internal/ocr/jobs/"+scan.JobID, nil, &jobDetail)
		if err == nil && jobDetail.Status != "" {
			scan.Status = jobDetail.Status
			scan.QualityScore = jobDetail.QualityScore
			scan.Enhanced = jobDetail.Enhanced
			scan.Reason = jobDetail.Reason
			scan.ReasonMessage = jobDetail.ReasonMessage

			// Simpan confidences tanpa nilai NIK
			confMap := make(map[string]float64)
			for k, v := range jobDetail.Fields {
				if v != nil {
					confMap[k] = v.Confidence
				}
			}
			confJSON, _ := json.Marshal(confMap)
			scan.FieldConfidences = string(confJSON)

			// Simpan seluruh data saran OCR untuk pencocokan saat konfirmasi
			rawFieldsJSON, _ := json.Marshal(jobDetail.Fields)
			scan.OCRDataRaw = string(rawFieldsJSON)
			scan.UpdatedAt = time.Now()

			_ = s.ktpScanRepo.Update(ctx, scan)
		}
	}

	// Buat DTO respon
	var fieldsDTO *dto.KTPFieldsDTO
	if scan.OCRDataRaw != "" {
		var fieldsMap map[string]*dto.KTPFieldItem
		if err := json.Unmarshal([]byte(scan.OCRDataRaw), &fieldsMap); err == nil {
			fieldsDTO = &dto.KTPFieldsDTO{
				NIK:           fieldsMap["nik"],
				Nama:          fieldsMap["nama"],
				TempatLahir:   fieldsMap["tempat_lahir"],
				TanggalLahir:  fieldsMap["tanggal_lahir"],
				JenisKelamin:  fieldsMap["jenis_kelamin"],
				Alamat:        fieldsMap["alamat"],
				RTRW:          fieldsMap["rt_rw"],
				KelDesa:       fieldsMap["kel_desa"],
				Kecamatan:     fieldsMap["kecamatan"],
				Agama:         fieldsMap["agama"],
				Kewarganegaraan: fieldsMap["kewarganegaraan"],
			}
		}
	}

	return &dto.KTPScanStatusResponse{
		ScanID:        scan.ID.String(),
		Status:        scan.Status,
		QualityScore:  scan.QualityScore,
		Enhanced:      scan.Enhanced,
		Reason:        scan.Reason,
		ReasonMessage: scan.ReasonMessage,
		Fields:        fieldsDTO,
	}, nil
}

func (s *ktpScanService) ConfirmKTPScan(ctx context.Context, scanID, userID uuid.UUID, req dto.KTPConfirmRequest) (*dto.ProfileResponse, error) {
	scan, err := s.ktpScanRepo.FindByID(ctx, scanID)
	if err != nil {
		return nil, err
	}

	if scan.UserID != userID {
		return nil, ErrUnauthorizedAccess
	}

	if scan.ConfirmedAt != nil {
		return nil, ErrScanAlreadyConfirmed
	}

	// Bandingkan field yang diedit user dari saran OCR
	var editedFields []string
	if scan.OCRDataRaw != "" {
		var ocrFields map[string]*dto.KTPFieldItem
		if err := json.Unmarshal([]byte(scan.OCRDataRaw), &ocrFields); err == nil {
			if ocrFields["nik"] != nil && ocrFields["nik"].Value != req.NIK {
				editedFields = append(editedFields, "nik")
			}
			if ocrFields["nama"] != nil && !strings.EqualFold(ocrFields["nama"].Value, req.FullName) {
				editedFields = append(editedFields, "nama")
			}
			if ocrFields["tanggal_lahir"] != nil && ocrFields["tanggal_lahir"].Value != req.BirthDate {
				editedFields = append(editedFields, "tanggal_lahir")
			}
		}
	}

	editedJSON, _ := json.Marshal(editedFields)
	scan.FieldsEditedByUser = string(editedJSON)
	scan.OCRDataRaw = "" // Sesuai PRD C.4.11 & UU PDP: data mentah OCR tidak disimpan permanen
	now := time.Now()
	scan.ConfirmedAt = &now
	scan.UpdatedAt = now
	_ = s.ktpScanRepo.Update(ctx, scan)

	// Simpan biodata terkonfirmasi ke persons
	profileReq := dto.ProfileRequest{
		FullName:    req.FullName,
		NIK:         req.NIK,
		BirthPlace:  req.BirthPlace,
		BirthDate:   req.BirthDate,
		Gender:      req.Gender,
		Religion:    req.Religion,
		Nationality: req.Nationality,
		Phone:       req.Phone,
	}
	profileResp, err := s.identitySvc.UpdateProfile(ctx, userID, profileReq)
	if err != nil {
		return nil, err
	}

	// Simpan alamat KTP terkonfirmasi
	req.Address.Type = "ktp"
	_, _ = s.identitySvc.SaveAddress(ctx, userID, req.Address)

	return profileResp, nil
}
