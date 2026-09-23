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
}

type UpdateStudyProgramRequest struct {
	FacultyID     uint   `json:"faculty_id" binding:"required"`
	Code          string `json:"code" binding:"required,min=2,max=20"`
	NIMCode       string `json:"nim_code" binding:"required,len=2"`
	Name          string `json:"name" binding:"required,min=3,max=100"`
	Degree        string `json:"degree" binding:"required,oneof=D3 S1 S2 S3"`
	Accreditation string `json:"accreditation" binding:"required"`
	IsActive      bool   `json:"is_active"`
}

type PatchStudyProgramRequest struct {
	FacultyID     *uint   `json:"faculty_id"`
	Code          *string `json:"code" binding:"omitempty,min=2,max=20"`
	NIMCode       *string `json:"nim_code" binding:"omitempty,len=2"`
	Name          *string `json:"name" binding:"omitempty,min=3,max=100"`
	Degree        *string `json:"degree" binding:"omitempty,oneof=D3 S1 S2 S3"`
	Accreditation *string `json:"accreditation"`
	IsActive      *bool   `json:"is_active"`
}

type BatchStudyProgramsRequest struct {
	Codes []string `json:"codes" binding:"required,min=1"`
}
