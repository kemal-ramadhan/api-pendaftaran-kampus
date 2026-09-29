package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/storage"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
)

var (
	ErrValidation           = errors.New("data yang dikirim tidak valid")
	ErrUnauthorizedAccess   = errors.New("tidak memiliki hak akses ke data ini")
	ErrFileTooLarge         = errors.New("ukuran file melebihi batas 2 MB")
	ErrInvalidFileType      = errors.New("tipe file hanya boleh JPG atau PNG")
	ErrPhotoUploadFailed    = errors.New("gagal mengunggah foto profil")
)

type IdentityService interface {
	GetProfile(ctx context.Context, targetUserID, callerUserID uuid.UUID, callerRoles []string) (*dto.ProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.ProfileRequest) (*dto.ProfileResponse, error)
	GetAddresses(ctx context.Context, userID uuid.UUID) ([]dto.AddressResponse, error)
	SaveAddress(ctx context.Context, userID uuid.UUID, req dto.AddressRequest) (*dto.AddressResponse, error)
	GetGuardians(ctx context.Context, userID uuid.UUID) ([]dto.GuardianResponse, error)
	SaveGuardian(ctx context.Context, userID uuid.UUID, req dto.GuardianRequest) (*dto.GuardianResponse, error)
	GetEducation(ctx context.Context, userID uuid.UUID) ([]dto.EducationHistoryResponse, error)
	SaveEducation(ctx context.Context, userID uuid.UUID, req dto.EducationHistoryRequest) (*dto.EducationHistoryResponse, error)
	UpdatePhoto(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, contentType string) (string, error)
	GetPersonDetail(ctx context.Context, targetUserID uuid.UUID, callerUserID uuid.UUID, callerRoles []string) (*dto.PersonDetailResponse, error)
	BatchGetPersons(ctx context.Context, userIDs []uuid.UUID) ([]dto.PersonDetailResponse, error)
	SearchNIK(ctx context.Context, nikHash string) (*dto.PersonDetailResponse, error)
}

type identityService struct {
	personRepo    repository.PersonRepository
	addressRepo   repository.AddressRepository
	guardianRepo  repository.GuardianRepository
	educationRepo repository.EducationRepository
	storageClient storage.Client
	encryptionKey []byte
	photoBucket   string
}

func NewIdentityService(
	personRepo repository.PersonRepository,
	addressRepo repository.AddressRepository,
	guardianRepo repository.GuardianRepository,
	educationRepo repository.EducationRepository,
	storageClient storage.Client,
	encryptionKey []byte,
	photoBucket string,
) IdentityService {
	if photoBucket == "" {
		photoBucket = "pmb-identity-photos"
	}
	return &identityService{
		personRepo:    personRepo,
		addressRepo:   addressRepo,
		guardianRepo:  guardianRepo,
		educationRepo: educationRepo,
		storageClient: storageClient,
		encryptionKey: encryptionKey,
		photoBucket:   photoBucket,
	}
}

func (s *identityService) shouldMaskNIK(targetUserID, callerUserID uuid.UUID, callerRoles []string) bool {
	if callerUserID == targetUserID {
		return false
	}
	if slices.Contains(callerRoles, "superadmin") || slices.Contains(callerRoles, "admin_pmb") {
		return false
	}
	return true
}

func (s *identityService) GetProfile(ctx context.Context, targetUserID, callerUserID uuid.UUID, callerRoles []string) (*dto.ProfileResponse, error) {
	person, err := s.personRepo.FindByUserID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	decryptedNIK, err := crypto.DecryptNIK(person.NIK, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("gagal mendekripsi NIK pengguna: %w", err)
	}

	displayNIK := decryptedNIK
	if s.shouldMaskNIK(targetUserID, callerUserID, callerRoles) {
		displayNIK = crypto.MaskNIK(decryptedNIK)
	}

	birthDateStr := ""
	if person.BirthDate != nil {
		birthDateStr = person.BirthDate.Format("2006-01-02")
	}

	photoURL := ""
	if person.PhotoObjectKey != "" && s.storageClient != nil {
		url, err := s.storageClient.PresignedGetObject(ctx, s.photoBucket, person.PhotoObjectKey, 15*time.Minute)
		if err == nil {
			photoURL = url
		}
	}

	return &dto.ProfileResponse{
		UserID:         person.UserID,
		FullName:       person.FullName,
		NIK:            displayNIK,
		BirthPlace:     person.BirthPlace,
		BirthDate:      birthDateStr,
		Gender:         person.Gender,
		Religion:       person.Religion,
		Nationality:    person.Nationality,
		Phone:          person.Phone,
		PhotoObjectKey: person.PhotoObjectKey,
		PhotoURL:       photoURL,
	}, nil
}

func (s *identityService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.ProfileRequest) (*dto.ProfileResponse, error) {
	cleanNIK := strings.TrimSpace(req.NIK)
	if len(cleanNIK) != 16 {
		return nil, fmt.Errorf("%w: NIK harus tepat 16 digit angka", ErrValidation)
	}

	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return nil, fmt.Errorf("%w: Format tanggal lahir harus YYYY-MM-DD", ErrValidation)
	}

	nikHash := crypto.HashNIK(cleanNIK)
	encryptedNIK, err := crypto.EncryptNIK(cleanNIK, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("gagal mengenkripsi NIK: %w", err)
	}

	nat := strings.TrimSpace(req.Nationality)
	if nat == "" {
		nat = "WNI"
	}

	person := &model.Person{
		UserID:      userID,
		FullName:    strings.TrimSpace(req.FullName),
		NIK:         encryptedNIK,
		NIKHash:     nikHash,
		BirthPlace:  strings.TrimSpace(req.BirthPlace),
		BirthDate:   &birthDate,
		Gender:      strings.ToUpper(strings.TrimSpace(req.Gender)),
		Religion:    strings.TrimSpace(req.Religion),
		Nationality: nat,
		Phone:       strings.TrimSpace(req.Phone),
		UpdatedAt:   time.Now(),
	}

	if err := s.personRepo.Save(ctx, person); err != nil {
		return nil, err
	}

	return s.GetProfile(ctx, userID, userID, []string{"calon_mahasiswa"})
}

func (s *identityService) GetAddresses(ctx context.Context, userID uuid.UUID) ([]dto.AddressResponse, error) {
	addrs, err := s.addressRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var res []dto.AddressResponse
	for _, a := range addrs {
		res = append(res, dto.AddressResponse{
			ID:         a.ID,
			UserID:     a.UserID,
			Type:       a.Type,
			Street:     a.Street,
			RT:         a.RT,
			RW:         a.RW,
			Village:    a.Village,
			District:   a.District,
			City:       a.City,
			Province:   a.Province,
			PostalCode: a.PostalCode,
		})
	}
	return res, nil
}

func (s *identityService) SaveAddress(ctx context.Context, userID uuid.UUID, req dto.AddressRequest) (*dto.AddressResponse, error) {
	addr := &model.Address{
		UserID:     userID,
		Type:       strings.ToLower(strings.TrimSpace(req.Type)),
		Street:     strings.TrimSpace(req.Street),
		RT:         strings.TrimSpace(req.RT),
		RW:         strings.TrimSpace(req.RW),
		Village:    strings.TrimSpace(req.Village),
		District:   strings.TrimSpace(req.District),
		City:       strings.TrimSpace(req.City),
		Province:   strings.TrimSpace(req.Province),
		PostalCode: strings.TrimSpace(req.PostalCode),
		UpdatedAt:  time.Now(),
	}

	if err := s.addressRepo.Save(ctx, addr); err != nil {
		return nil, err
	}

	saved, err := s.addressRepo.FindByUserIDAndType(ctx, userID, addr.Type)
	if err != nil {
		return nil, err
	}

	return &dto.AddressResponse{
		ID:         saved.ID,
		UserID:     saved.UserID,
		Type:       saved.Type,
		Street:     saved.Street,
		RT:         saved.RT,
		RW:         saved.RW,
		Village:    saved.Village,
		District:   saved.District,
		City:       saved.City,
		Province:   saved.Province,
		PostalCode: saved.PostalCode,
	}, nil
}

func (s *identityService) GetGuardians(ctx context.Context, userID uuid.UUID) ([]dto.GuardianResponse, error) {
	list, err := s.guardianRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var res []dto.GuardianResponse
	for _, g := range list {
		res = append(res, dto.GuardianResponse{
			ID:          g.ID,
			UserID:      g.UserID,
			Relation:    g.Relation,
			Name:        g.Name,
			Occupation:  g.Occupation,
			IncomeRange: g.IncomeRange,
			Phone:       g.Phone,
		})
	}
	return res, nil
}

func (s *identityService) SaveGuardian(ctx context.Context, userID uuid.UUID, req dto.GuardianRequest) (*dto.GuardianResponse, error) {
	g := &model.Guardian{
		UserID:      userID,
		Relation:    strings.ToLower(strings.TrimSpace(req.Relation)),
		Name:        strings.TrimSpace(req.Name),
		Occupation:  strings.TrimSpace(req.Occupation),
		IncomeRange: strings.TrimSpace(req.IncomeRange),
		Phone:       strings.TrimSpace(req.Phone),
		UpdatedAt:   time.Now(),
	}

	if err := s.guardianRepo.Save(ctx, g); err != nil {
		return nil, err
	}

	saved, err := s.guardianRepo.FindByUserIDAndRelation(ctx, userID, g.Relation)
	if err != nil {
		return nil, err
	}

	return &dto.GuardianResponse{
		ID:          saved.ID,
		UserID:      saved.UserID,
		Relation:    saved.Relation,
		Name:        saved.Name,
		Occupation:  saved.Occupation,
		IncomeRange: saved.IncomeRange,
		Phone:       saved.Phone,
	}, nil
}

func (s *identityService) GetEducation(ctx context.Context, userID uuid.UUID) ([]dto.EducationHistoryResponse, error) {
	list, err := s.educationRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var res []dto.EducationHistoryResponse
	for _, e := range list {
		res = append(res, dto.EducationHistoryResponse{
			ID:             e.ID,
			UserID:         e.UserID,
			Level:          e.Level,
			SchoolName:     e.SchoolName,
			NPSN:           e.NPSN,
			Major:          e.Major,
			GraduationYear: e.GraduationYear,
			ReportAverage:  e.ReportAverage,
		})
	}
	return res, nil
}

func (s *identityService) SaveEducation(ctx context.Context, userID uuid.UUID, req dto.EducationHistoryRequest) (*dto.EducationHistoryResponse, error) {
	e := &model.EducationHistory{
		UserID:         userID,
		Level:          strings.ToUpper(strings.TrimSpace(req.Level)),
		SchoolName:     strings.TrimSpace(req.SchoolName),
		NPSN:           strings.TrimSpace(req.NPSN),
		Major:          strings.TrimSpace(req.Major),
		GraduationYear: req.GraduationYear,
		ReportAverage:  req.ReportAverage,
		UpdatedAt:      time.Now(),
	}

	if err := s.educationRepo.Save(ctx, e); err != nil {
		return nil, err
	}

	return &dto.EducationHistoryResponse{
		ID:             e.ID,
		UserID:         e.UserID,
		Level:          e.Level,
		SchoolName:     e.SchoolName,
		NPSN:           e.NPSN,
		Major:          e.Major,
		GraduationYear: e.GraduationYear,
		ReportAverage:  e.ReportAverage,
	}, nil
}

func (s *identityService) UpdatePhoto(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, contentType string) (string, error) {
	if size > 2*1024*1024 {
		return "", ErrFileTooLarge
	}

	// Sniff content type jika storageClient mendukung
	allowedTypes := []string{"image/jpeg", "image/png"}
	var validatedReader io.Reader = reader
	var sniffedType = contentType

	if s.storageClient != nil {
		vr, st, err := s.storageClient.SniffAndValidate(reader, allowedTypes)
		if err != nil {
			return "", ErrInvalidFileType
		}
		validatedReader = vr
		sniffedType = st

		_ = s.storageClient.EnsureBucket(ctx, s.photoBucket)
	} else {
		if !slices.Contains(allowedTypes, contentType) {
			return "", ErrInvalidFileType
		}
	}

	ext := "jpg"
	if strings.Contains(sniffedType, "png") {
		ext = "png"
	}
	objectKey := fmt.Sprintf("photos/%s/%d.%s", userID.String(), time.Now().Unix(), ext)

	if s.storageClient != nil {
		err := s.storageClient.Upload(ctx, s.photoBucket, objectKey, validatedReader, size, sniffedType)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrPhotoUploadFailed, err)
		}
	}

	if err := s.personRepo.UpdatePhoto(ctx, userID, objectKey); err != nil {
		return "", err
	}

	photoURL := objectKey
	if s.storageClient != nil {
		url, err := s.storageClient.PresignedGetObject(ctx, s.photoBucket, objectKey, 24*time.Hour)
		if err == nil {
			photoURL = url
		}
	}

	return photoURL, nil
}

func (s *identityService) GetPersonDetail(ctx context.Context, targetUserID uuid.UUID, callerUserID uuid.UUID, callerRoles []string) (*dto.PersonDetailResponse, error) {
	person, err := s.personRepo.GetDetailWithRelations(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	decryptedNIK, err := crypto.DecryptNIK(person.NIK, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("gagal dekripsi NIK: %w", err)
	}

	displayNIK := decryptedNIK
	if s.shouldMaskNIK(targetUserID, callerUserID, callerRoles) {
		displayNIK = crypto.MaskNIK(decryptedNIK)
	}

	birthDateStr := ""
	if person.BirthDate != nil {
		birthDateStr = person.BirthDate.Format("2006-01-02")
	}

	photoURL := ""
	if person.PhotoObjectKey != "" && s.storageClient != nil {
		url, err := s.storageClient.PresignedGetObject(ctx, s.photoBucket, person.PhotoObjectKey, 15*time.Minute)
		if err == nil {
			photoURL = url
		}
	}

	profileResp := dto.ProfileResponse{
		UserID:         person.UserID,
		FullName:       person.FullName,
		NIK:            displayNIK,
		BirthPlace:     person.BirthPlace,
		BirthDate:      birthDateStr,
		Gender:         person.Gender,
		Religion:       person.Religion,
		Nationality:    person.Nationality,
		Phone:          person.Phone,
		PhotoObjectKey: person.PhotoObjectKey,
		PhotoURL:       photoURL,
	}

	var addrs []dto.AddressResponse
	for _, a := range person.Addresses {
		addrs = append(addrs, dto.AddressResponse{
			ID:         a.ID,
			UserID:     a.UserID,
			Type:       a.Type,
			Street:     a.Street,
			RT:         a.RT,
			RW:         a.RW,
			Village:    a.Village,
			District:   a.District,
			City:       a.City,
			Province:   a.Province,
			PostalCode: a.PostalCode,
		})
	}

	var guardians []dto.GuardianResponse
	for _, g := range person.Guardians {
		guardians = append(guardians, dto.GuardianResponse{
			ID:          g.ID,
			UserID:      g.UserID,
			Relation:    g.Relation,
			Name:        g.Name,
			Occupation:  g.Occupation,
			IncomeRange: g.IncomeRange,
			Phone:       g.Phone,
		})
	}

	var edus []dto.EducationHistoryResponse
	for _, e := range person.EducationHistories {
		edus = append(edus, dto.EducationHistoryResponse{
			ID:             e.ID,
			UserID:         e.UserID,
			Level:          e.Level,
			SchoolName:     e.SchoolName,
			NPSN:           e.NPSN,
			Major:          e.Major,
			GraduationYear: e.GraduationYear,
			ReportAverage:  e.ReportAverage,
		})
	}

	return &dto.PersonDetailResponse{
		Profile:            profileResp,
		Addresses:          addrs,
		Guardians:          guardians,
		EducationHistories: edus,
	}, nil
}

func (s *identityService) BatchGetPersons(ctx context.Context, userIDs []uuid.UUID) ([]dto.PersonDetailResponse, error) {
	persons, err := s.personRepo.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	var result []dto.PersonDetailResponse
	for _, p := range persons {
		decryptedNIK, err := crypto.DecryptNIK(p.NIK, s.encryptionKey)
		if err != nil {
			decryptedNIK = "****************"
		}

		birthDateStr := ""
		if p.BirthDate != nil {
			birthDateStr = p.BirthDate.Format("2006-01-02")
		}

		profile := dto.ProfileResponse{
			UserID:         p.UserID,
			FullName:       p.FullName,
			NIK:            decryptedNIK,
			BirthPlace:     p.BirthPlace,
			BirthDate:      birthDateStr,
			Gender:         p.Gender,
			Religion:       p.Religion,
			Nationality:    p.Nationality,
			Phone:          p.Phone,
			PhotoObjectKey: p.PhotoObjectKey,
		}

		var addrs []dto.AddressResponse
		for _, a := range p.Addresses {
			addrs = append(addrs, dto.AddressResponse{
				ID: a.ID, UserID: a.UserID, Type: a.Type, Street: a.Street,
				RT: a.RT, RW: a.RW, Village: a.Village, District: a.District,
				City: a.City, Province: a.Province, PostalCode: a.PostalCode,
			})
		}

		var guardians []dto.GuardianResponse
		for _, g := range p.Guardians {
			guardians = append(guardians, dto.GuardianResponse{
				ID: g.ID, UserID: g.UserID, Relation: g.Relation,
				Name: g.Name, Occupation: g.Occupation, IncomeRange: g.IncomeRange, Phone: g.Phone,
			})
		}

		var edus []dto.EducationHistoryResponse
		for _, e := range p.EducationHistories {
			edus = append(edus, dto.EducationHistoryResponse{
				ID: e.ID, UserID: e.UserID, Level: e.Level, SchoolName: e.SchoolName,
				NPSN: e.NPSN, Major: e.Major, GraduationYear: e.GraduationYear,
				ReportAverage:  e.ReportAverage,
			})
		}

		result = append(result, dto.PersonDetailResponse{
			Profile:            profile,
			Addresses:          addrs,
			Guardians:          guardians,
			EducationHistories: edus,
		})
	}

	return result, nil
}

func (s *identityService) SearchNIK(ctx context.Context, nikHash string) (*dto.PersonDetailResponse, error) {
	person, err := s.personRepo.FindByNIKHash(ctx, nikHash)
	if err != nil {
		return nil, err
	}
	return s.GetPersonDetail(ctx, person.UserID, person.UserID, []string{"superadmin"})
}
