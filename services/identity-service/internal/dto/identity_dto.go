package dto

import (
	"github.com/google/uuid"
)

// ProfileRequest payload untuk update profile (/me/profile)
type ProfileRequest struct {
	FullName    string `json:"full_name" binding:"required"`
	NIK         string `json:"nik" binding:"required,len=16,numeric"`
	BirthPlace  string `json:"birth_place" binding:"required"`
	BirthDate   string `json:"birth_date" binding:"required"` // Format YYYY-MM-DD
	Gender      string `json:"gender" binding:"required,oneof=L P"`
	Religion    string `json:"religion" binding:"required"`
	Nationality string `json:"nationality"`
	Phone       string `json:"phone" binding:"required"`
}

// ProfileResponse respons profil pengguna
type ProfileResponse struct {
	UserID         uuid.UUID `json:"user_id"`
	FullName       string    `json:"full_name"`
	NIK            string    `json:"nik"` // Masked atau unmasked sesuai otorisasi caller
	BirthPlace     string    `json:"birth_place"`
	BirthDate      string    `json:"birth_date"`
	Gender         string    `json:"gender"`
	Religion       string    `json:"religion"`
	Nationality    string    `json:"nationality"`
	Phone          string    `json:"phone"`
	PhotoObjectKey string    `json:"photo_object_key,omitempty"`
	PhotoURL       string    `json:"photo_url,omitempty"`
}

// AddressRequest payload alamat (ktp atau domisili)
type AddressRequest struct {
	Type       string `json:"type" binding:"required,oneof=ktp domisili"`
	Street     string `json:"street" binding:"required"`
	RT         string `json:"rt" binding:"required"`
	RW         string `json:"rw" binding:"required"`
	Village    string `json:"village" binding:"required"`
	District   string `json:"district" binding:"required"`
	City       string `json:"city" binding:"required"`
	Province   string `json:"province" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
}

// AddressResponse representasi alamat
type AddressResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Type       string    `json:"type"`
	Street     string    `json:"street"`
	RT         string    `json:"rt"`
	RW         string    `json:"rw"`
	Village    string    `json:"village"`
	District   string    `json:"district"`
	City       string    `json:"city"`
	Province   string    `json:"province"`
	PostalCode string    `json:"postal_code"`
}

// GuardianRequest payload orang tua / wali
type GuardianRequest struct {
	Relation    string `json:"relation" binding:"required,oneof=ayah ibu wali"`
	Name        string `json:"name" binding:"required"`
	Occupation  string `json:"occupation"`
	IncomeRange string `json:"income_range"`
	Phone       string `json:"phone" binding:"required"`
}

// GuardianResponse representasi orang tua / wali
type GuardianResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Relation    string    `json:"relation"`
	Name        string    `json:"name"`
	Occupation  string    `json:"occupation"`
	IncomeRange string    `json:"income_range"`
	Phone       string    `json:"phone"`
}

// EducationHistoryRequest payload riwayat pendidikan
type EducationHistoryRequest struct {
	Level          string `json:"level" binding:"required"` // SMA, SMK, MA, D3, dll
	SchoolName     string `json:"school_name" binding:"required"`
	NPSN           string `json:"npsn"`
	Major          string `json:"major"`
	GraduationYear int    `json:"graduation_year" binding:"required,min=1950,max=2100"`
}

// EducationHistoryResponse representasi riwayat pendidikan
type EducationHistoryResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Level          string    `json:"level"`
	SchoolName     string    `json:"school_name"`
	NPSN           string    `json:"npsn"`
	Major          string    `json:"major"`
	GraduationYear int       `json:"graduation_year"`
}

// CompletenessResponse respons kelengkapan data biodata (/me/completeness)
type CompletenessResponse struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

// KTPFieldItem representasi field individu hasil pembacaan OCR
type KTPFieldItem struct {
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	NeedsReview bool    `json:"needs_review"`
}

// KTPFieldsDTO daftar field yang diekstrak dari kartu KTP
type KTPFieldsDTO struct {
	NIK           *KTPFieldItem `json:"nik,omitempty"`
	Nama          *KTPFieldItem `json:"nama,omitempty"`
	TempatLahir   *KTPFieldItem `json:"tempat_lahir,omitempty"`
	TanggalLahir  *KTPFieldItem `json:"tanggal_lahir,omitempty"`
	JenisKelamin  *KTPFieldItem `json:"jenis_kelamin,omitempty"`
	Alamat        *KTPFieldItem `json:"alamat,omitempty"`
	RTRW          *KTPFieldItem `json:"rt_rw,omitempty"`
	KelDesa       *KTPFieldItem `json:"kel_desa,omitempty"`
	Kecamatan     *KTPFieldItem `json:"kecamatan,omitempty"`
	Agama         *KTPFieldItem `json:"agama,omitempty"`
	Kewarganegaraan *KTPFieldItem `json:"kewarganegaraan,omitempty"`
}

// KTPScanInitiateResponse respons inisiasi pemindaian KTP (/me/ktp-scan)
type KTPScanInitiateResponse struct {
	ScanID string `json:"scan_id"`
	Status string `json:"status"`
}

// KTPScanStatusResponse respons status pemindaian KTP (/me/ktp-scan/:scan_id)
type KTPScanStatusResponse struct {
	ScanID        string        `json:"scan_id"`
	Status        string        `json:"status"` // processing, completed, retake_required, failed
	QualityScore  float64       `json:"quality_score"`
	Enhanced      bool          `json:"enhanced"`
	Reason        string        `json:"reason,omitempty"`
	ReasonMessage string        `json:"reason_message,omitempty"`
	Fields        *KTPFieldsDTO `json:"fields,omitempty"`
}

// KTPConfirmRequest konfirmasi data hasil OCR yang diedit/diverifikasi user
type KTPConfirmRequest struct {
	FullName    string         `json:"full_name" binding:"required"`
	NIK         string         `json:"nik" binding:"required,len=16,numeric"`
	BirthPlace  string         `json:"birth_place" binding:"required"`
	BirthDate   string         `json:"birth_date" binding:"required"` // YYYY-MM-DD
	Gender      string         `json:"gender" binding:"required,oneof=L P"`
	Religion    string         `json:"religion" binding:"required"`
	Nationality string         `json:"nationality"`
	Phone       string         `json:"phone"`
	Address     AddressRequest `json:"address" binding:"required"`
}

// PersonDetailResponse detail lengkap orang untuk admin & internal
type PersonDetailResponse struct {
	Profile            ProfileResponse            `json:"profile"`
	Addresses          []AddressResponse          `json:"addresses"`
	Guardians          []GuardianResponse         `json:"guardians"`
	EducationHistories []EducationHistoryResponse `json:"education_histories"`
	Completeness       CompletenessResponse       `json:"completeness"`
}

// BatchPersonsRequest request batch internal
type BatchPersonsRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}
