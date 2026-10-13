package dto

import "time"

// Faculty DTOs
type FacultyResponse struct {
	ID        uint      `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateFacultyRequest struct {
	Code     string `json:"code" binding:"required,min=2,max=20"`
	Name     string `json:"name" binding:"required,min=3,max=100"`
	IsActive *bool  `json:"is_active"`
}

type UpdateFacultyRequest struct {
	Code     string `json:"code" binding:"required,min=2,max=20"`
	Name     string `json:"name" binding:"required,min=3,max=100"`
	IsActive bool   `json:"is_active"`
}

type PatchFacultyRequest struct {
	Code     *string `json:"code" binding:"omitempty,min=2,max=20"`
	Name     *string `json:"name" binding:"omitempty,min=3,max=100"`
	IsActive *bool   `json:"is_active"`
}

// Study Program DTOs
type StudyProgramResponse struct {
	ID            uint      `json:"id"`
	FacultyID     uint      `json:"faculty_id"`
	FacultyCode   string    `json:"faculty_code,omitempty"`
	FacultyName   string    `json:"faculty_name,omitempty"`
	Code          string    `json:"code"`
	NIMCode       string    `json:"nim_code"`
	Name          string    `json:"name"`
	Degree        string    `json:"degree"`
	Accreditation string    `json:"accreditation"`
	IsActive      bool      `json:"is_active"`
	AdmissionOpen bool      `json:"admission_open"`
	AdmissionStatusLabel string `json:"admission_status_label"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateStudyProgramRequest struct {
	FacultyID     uint   `json:"faculty_id" binding:"required"`
	Code          string `json:"code" binding:"required,min=2,max=20"`
	NIMCode       string `json:"nim_code" binding:"required,len=2"`
	Name          string `json:"name" binding:"required,min=3,max=100"`
	Degree        string `json:"degree" binding:"required,oneof=D3 S1 S2 S3"`
	Accreditation string `json:"accreditation" binding:"required"`
	IsActive      *bool  `json:"is_active"`
	AdmissionOpen *bool  `json:"admission_open"`
}

type UpdateStudyProgramRequest struct {
	FacultyID     uint   `json:"faculty_id" binding:"required"`
	Code          string `json:"code" binding:"required,min=2,max=20"`
	NIMCode       string `json:"nim_code" binding:"required,len=2"`
	Name          string `json:"name" binding:"required,min=3,max=100"`
	Degree        string `json:"degree" binding:"required,oneof=D3 S1 S2 S3"`
	Accreditation string `json:"accreditation" binding:"required"`
	IsActive      bool   `json:"is_active"`
	AdmissionOpen bool   `json:"admission_open"`
}

type PatchStudyProgramRequest struct {
	FacultyID     *uint   `json:"faculty_id"`
	Code          *string `json:"code" binding:"omitempty,min=2,max=20"`
	NIMCode       *string `json:"nim_code" binding:"omitempty,len=2"`
	Name          *string `json:"name" binding:"omitempty,min=3,max=100"`
	Degree        *string `json:"degree" binding:"omitempty,oneof=D3 S1 S2 S3"`
	Accreditation *string `json:"accreditation"`
	IsActive      *bool   `json:"is_active"`
	AdmissionOpen *bool   `json:"admission_open"`
}

type PatchAdmissionOpenRequest struct {
	AdmissionOpen bool `json:"admission_open"`
}

type BatchStudyProgramsRequest struct {
	Codes []string `json:"codes" binding:"required,min=1"`
}

// Institution Profile DTOs
type BankAccountDTO struct {
	BankName      string `json:"bank_name" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
	AccountName   string `json:"account_name" binding:"required"`
	IsPrimary     bool   `json:"is_primary"`
}

type InstitutionProfileResponse struct {
	ID                  uint             `json:"id"`
	FoundationName      string           `json:"foundation_name"`
	InstitutionName     string           `json:"institution_name"`
	DocumentCode        string           `json:"document_code"`
	Address             string           `json:"address"`
	City                string           `json:"city"`
	Email               string           `json:"email"`
	Phone               string           `json:"phone"`
	Hotline             string           `json:"hotline"`
	ServiceHours        string           `json:"service_hours"`
	LogoURL             string           `json:"logo_url"`
	CommitteeChairName  string           `json:"committee_chair_name"`
	CommitteeChairTitle string           `json:"committee_chair_title"`
	CommitteeChairNIP   string           `json:"committee_chair_nip"`
	BankAccounts        []BankAccountDTO `json:"bank_accounts"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

type UpdateInstitutionProfileRequest struct {
	FoundationName      string           `json:"foundation_name" binding:"required"`
	InstitutionName     string           `json:"institution_name" binding:"required"`
	DocumentCode        string           `json:"document_code" binding:"required"`
	Address             string           `json:"address"`
	City                string           `json:"city" binding:"required"`
	Email               string           `json:"email"`
	Phone               string           `json:"phone"`
	Hotline             string           `json:"hotline"`
	ServiceHours        string           `json:"service_hours"`
	LogoURL             string           `json:"logo_url"`
	CommitteeChairName  string           `json:"committee_chair_name"`
	CommitteeChairTitle string           `json:"committee_chair_title"`
	CommitteeChairNIP   string           `json:"committee_chair_nip"`
	BankAccounts        []BankAccountDTO `json:"bank_accounts"`
}

// Programs Overview DTOs (Halaman 24 / Bagian 2.8)
type ProgramOverviewItem struct {
	ID                       uint   `json:"id"`
	Code                     string `json:"code"`
	NIMCode                  string `json:"nim_code"`
	Name                     string `json:"name"`
	FacultyCode              string `json:"faculty_code"`
	FacultyName              string `json:"faculty_name"`
	Degree                   string `json:"degree"`
	Accreditation            string `json:"accreditation"`
	AdmissionOpen            bool   `json:"admission_open"`
	AdmissionStatusLabel     string `json:"admission_status_label"` // "Buka" atau "Tutup"
	EstimatedTuitionFee      int64  `json:"estimated_tuition_fee"`
	EstimatedTuitionFeeLabel string `json:"estimated_tuition_fee_label"`
}

type ProgramsOverviewResponse struct {
	ActiveFacultiesCount int64                 `json:"active_faculties_count"`
	TotalProgramsCount   int64                 `json:"total_programs_count"`
	OpenProgramsCount    int64                 `json:"open_programs_count"`
	NIMFormat            string                `json:"nim_format"`
	Programs             []ProgramOverviewItem `json:"programs"`
}
