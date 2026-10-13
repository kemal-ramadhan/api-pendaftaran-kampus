package service

import (
	"time"

	"github.com/google/uuid"
)

// HeaderSummaryResponse DTO untuk GET /admission/admin/header-summary
type HeaderSummaryResponse struct {
	PeriodID             uint   `json:"period_id"`
	PeriodName           string `json:"period_name"`
	TotalApplicants      int64  `json:"total_applicants"`
	TotalApplicantsLabel string `json:"total_applicants_label"`
	InstitutionName      string `json:"institution_name"`
	CommitteeBadge       string `json:"committee_badge"`
}

// OverviewCardKPI DTO kartu metrik di overview
type OverviewCardKPI struct {
	TotalApplicants            *int64   `json:"total_applicants"`
	PendingDocumentVerification *int64  `json:"pending_document_verification"`
	TotalRevenue               int64    `json:"total_revenue"`
	TotalRevenueLabel          string   `json:"total_revenue_label"`
	PassedApplicants           *int64   `json:"passed_applicants"`
	PassingRatePercentage      *float64 `json:"passing_rate_percentage"`
}

// ProgramProportionItem DTO donat proporsi peminat
type ProgramProportionItem struct {
	ProgramCode string  `json:"program_code"`
	Label       string  `json:"label"`
	Count       int64   `json:"count"`
	Percentage  float64 `json:"percentage"`
}

// FunnelStepItem DTO tahap funnel konversi 6 tahap
type FunnelStepItem struct {
	StepNo     int     `json:"step_no"`
	Name       string  `json:"name"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

// QuotaDistributionItem DTO keterisian kuota per prodi
type QuotaDistributionItem struct {
	ProgramCode      string  `json:"program_code"`
	ProgramName      string  `json:"program_name"`
	Quota            int     `json:"quota"`
	Filled           int     `json:"filled"`
	FilledPercentage float64 `json:"filled_percentage"`
}

// OverviewResponse DTO untuk GET /admission/admin/overview
type OverviewResponse struct {
	PeriodID              uint                    `json:"period_id"`
	PeriodName            string                  `json:"period_name"`
	KPIs                  OverviewCardKPI         `json:"kpis"`
	ProgramProportions    []ProgramProportionItem `json:"program_proportions,omitempty"`
	ConversionFunnel      []FunnelStepItem        `json:"conversion_funnel,omitempty"`
	OverallConversionRate *float64                `json:"overall_conversion_rate,omitempty"`
	QuotaDistribution     []QuotaDistributionItem `json:"quota_distribution,omitempty"`
	RecentActivities      []interface{}           `json:"recent_activities,omitempty"`
}

// DocumentQueueItem DTO baris antrean verifikasi berkas
type DocumentQueueItem struct {
	ApplicationID          uuid.UUID `json:"application_id"`
	RegistrationNumber     string    `json:"registration_number"`
	MaskedNIK              string    `json:"masked_nik"`
	FullName               string    `json:"full_name"`
	SchoolOrigin           string    `json:"school_origin"`
	ProgramCode            string    `json:"program_code"`
	ProgramName            string    `json:"program_name"`
	DocumentsSummary       string    `json:"documents_summary"` // "x/5 sah"
	DocumentsValidCount    int       `json:"documents_valid_count"`
	TotalRequiredDocuments int       `json:"total_required_documents"`
	Status                 string    `json:"status"`
	StatusLabel            string    `json:"status_label"`
	SubmittedAt            *time.Time `json:"submitted_at,omitempty"`
}

// DocumentQueueCounts DTO jumlah dokumen per tab status
type DocumentQueueCounts struct {
	All         int64 `json:"all"`
	Waiting     int64 `json:"waiting"`
	Verified    int64 `json:"verified"`
	Revision    int64 `json:"revision"`
	NotUploaded int64 `json:"not_uploaded"`
}

// DocumentQueueResponse DTO untuk GET /admission/admin/document-queue
type DocumentQueueResponse struct {
	Items []DocumentQueueItem `json:"items"`
	Meta  struct {
		Total  int64               `json:"total"`
		Page   int                 `json:"page"`
		Limit  int                 `json:"limit"`
		Counts DocumentQueueCounts `json:"counts"`
	} `json:"meta"`
}

// ApplicationDocumentDetailItem DTO dokumen modal verifikasi
type ApplicationDocumentDetailItem struct {
	DocumentType string     `json:"document_type"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	FileURL      string     `json:"file_url"`
	PreviewURL   string     `json:"preview_url"`
	Status       string     `json:"status"`
	Notes        string     `json:"notes,omitempty"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	VerifiedBy   *uuid.UUID `json:"verified_by,omitempty"`
}

// ApplicationDocumentsModalResponse DTO untuk GET /admission/admin/applications/:id/documents
type ApplicationDocumentsModalResponse struct {
	ApplicationID      uuid.UUID                       `json:"application_id"`
	RegistrationNumber string                          `json:"registration_number"`
	FullName           string                          `json:"full_name"`
	ProgramName        string                          `json:"program_name"`
	SchoolOrigin       string                          `json:"school_origin"`
	Status             string                          `json:"status"`
	StatusLabel        string                          `json:"status_label"`
	VerifiedCount      int                             `json:"verified_count"`
	TotalCount         int                             `json:"total_count"`
	SummaryLabel       string                          `json:"summary_label"` // "x/5 disetujui"
	NextApplicationID  *uuid.UUID                      `json:"next_application_id,omitempty"`
	Documents          []ApplicationDocumentDetailItem `json:"documents"`
}

// FacultyPerformanceItem DTO kinerja antar fakultas
type FacultyPerformanceItem struct {
	FacultyCode      string   `json:"faculty_code"`
	FacultyName      string   `json:"faculty_name"`
	Capacity         int      `json:"capacity"`
	ApplicantsCount  int      `json:"applicants_count"`
	PassedCount      int      `json:"passed_count"`
	FilledPercentage float64  `json:"filled_percentage"`
	ProgramNames     []string `json:"program_names"`
}

// ProgramAverageScoreItem DTO rata-rata skor CBT & rapor per prodi
type ProgramAverageScoreItem struct {
	ProgramCode    string  `json:"program_code"`
	ProgramName    string  `json:"program_name"`
	AverageCBT     float64 `json:"average_cbt"`
	AverageReport  float64 `json:"average_report"`
	ApplicantsCount int    `json:"applicants_count"`
}

// InstitutionDashboardResponse DTO untuk GET /admission/admin/institution-dashboard
type InstitutionDashboardResponse struct {
	PeriodID            uint                      `json:"period_id"`
	Level               string                    `json:"level"` // university | faculty | program
	Code                string                    `json:"code,omitempty"`
	TotalQuota          int                       `json:"total_quota"`
	FilledQuota         int                       `json:"filled_quota"`
	CBTPassingRate      float64                   `json:"cbt_passing_rate"`
	TightnessRatio      string                    `json:"tightness_ratio"` // mis. "1:3.4"
	TotalRevenue        int64                     `json:"total_revenue"`
	TotalRevenueLabel   string                    `json:"total_revenue_label"`
	FacultyComparisons  []FacultyPerformanceItem  `json:"faculty_comparisons,omitempty"`
	ProgramAverages     []ProgramAverageScoreItem `json:"program_averages,omitempty"`
}

// ReportPreviewItem DTO pratinjau tabel terpadu laporan
type ReportPreviewItem struct {
	RegistrationNumber string  `json:"registration_number"`
	FullName           string  `json:"full_name"`
	ProgramName        string  `json:"program_name"`
	FormStatus         string  `json:"form_status"`
	CBTScore           *float64 `json:"cbt_score,omitempty"`
	LoAStatus          string  `json:"loa_status"`
	TuitionStatus      string  `json:"tuition_status"`
}

// ReportPreviewResponse DTO untuk GET /admission/admin/reports/preview
type ReportPreviewResponse struct {
	Items []ReportPreviewItem `json:"items"`
	Meta  struct {
		Total int64 `json:"total"`
	} `json:"meta"`
}

// CamabaDashboardV2Response DTO untuk Portal Camaba v2 (PRD 3.1)
type CamabaDashboardV2Response struct {
	ApplicationID    *uuid.UUID          `json:"application_id,omitempty"`
	RegistrationNo   string              `json:"registration_no,omitempty"`
	Status           string              `json:"status"`
	StatusLabel      string              `json:"status_label"`
	CompletedSteps   int                 `json:"completed_steps"`
	TotalSteps       int                 `json:"total_steps"` // Selalu 6
	Steps            []CamabaStepItem    `json:"steps"`
	CurrentAction    *CamabaActionItem   `json:"current_action,omitempty"`
	Agenda           []CamabaAgendaItem  `json:"agenda"`
	SummaryCards     []CamabaSummaryCard `json:"summary_cards"`
	ServiceCenter    *CamabaServiceCenter `json:"service_center,omitempty"`
}

type CamabaStepItem struct {
	Key         string `json:"key"` // profile, path, payment, documents, selection, reregistration
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	IsCompleted bool   `json:"is_completed"`
	IsCurrent   bool   `json:"is_current"`
	State       string `json:"state"` // done, active, upcoming, locked
}

type CamabaActionItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionURL   string `json:"action_url"`
	ButtonLabel string `json:"button_label"`
}

type CamabaAgendaItem struct {
	No          int    `json:"no"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DateLabel   string `json:"date_label"`
	State       string `json:"state"` // active, upcoming, done
}

type CamabaSummaryCard struct {
	Key         string `json:"key"` // documents, payment, exam
	Title       string `json:"title"`
	Value       string `json:"value"`
	StatusLabel string `json:"status_label"`
	ActionLabel string `json:"action_label"`
	ActionURL   string `json:"action_url"`
}

type CamabaServiceCenter struct {
	InstitutionName string `json:"institution_name"`
	Address         string `json:"address"`
	ServiceHours    string `json:"service_hours"`
	Hotline         string `json:"hotline"`
	WhatsApp        string `json:"whatsapp"`
	Email           string `json:"email"`
}

// AdmissionOptionsResponse DTO untuk GET /admission/options (PRD 3.3)
type AdmissionOptionsResponse struct {
	PeriodID           uint                   `json:"period_id"`
	PeriodName         string                 `json:"period_name"`
	ChoicesStatusLabel string                 `json:"choices_status_label"`
	SecondChoiceNote   string                 `json:"second_choice_note"`
	Paths              []AdmissionPathOption  `json:"paths"`
	StudyPrograms      []StudyProgramOption   `json:"study_programs"`
}

type AdmissionPathOption struct {
	ID                   uint   `json:"id"`
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	RegistrationFee      int64  `json:"registration_fee"`
	RegistrationFeeLabel string `json:"registration_fee_label"`
	RequiresExam         bool   `json:"requires_exam"`
	IsActive             bool   `json:"is_active"`
}

type StudyProgramOption struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	FacultyName   string    `json:"faculty_name"`
	Degree        string    `json:"degree"`
	AdmissionOpen bool      `json:"admission_open"`
}
