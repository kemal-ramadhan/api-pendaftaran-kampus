package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/exporter"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/scoring-service/internal/repository"
)

var (
	ErrComponentNotFound       = errors.New("komponen nilai tidak ditemukan")
	ErrDuplicateComponent      = errors.New("DUPLICATE_COMPONENT: kode komponen nilai sudah ada pada jalur ini")
	ErrInvalidWeightSum        = errors.New("INVALID_WEIGHT_SUM: total persentase bobot komponen aktif pada jalur pendaftaran harus tepat 100%")
	ErrScoresLocked            = errors.New("SCORES_LOCKED: nilai pendaftar pada gelombang dan program studi ini telah dikunci dan tidak dapat diubah lagi")
	ErrForbiddenExaminer       = errors.New("FORBIDDEN: Anda bukan penguji yang ditugaskan pada sesi ujian peserta ini")
	ErrInvalidScoreRange       = errors.New("skor harus bernilai positif dan tidak melebihi skor maksimal")
	ErrApplicationNotFound     = errors.New("data pendaftaran tidak ditemukan")
	ErrForbiddenProgramAccess  = errors.New("FORBIDDEN: Anda tidak memiliki hak akses ke rekapitulasi program studi ini")
	ErrWeightExceeds100        = errors.New("INVALID_WEIGHT_SUM: total persentase bobot tidak boleh melebihi 100%")
)

type ScoringService interface {
	// Komponen Nilai
	CreateComponent(ctx context.Context, comp *model.ScoreComponent) (*model.ScoreComponent, error)
	UpdateComponent(ctx context.Context, comp *model.ScoreComponent) (*model.ScoreComponent, error)
	DeleteComponent(ctx context.Context, id uuid.UUID) error
	GetComponentsByPathID(ctx context.Context, pathID uuid.UUID) ([]model.ScoreComponent, error)
	ValidatePathWeightTotal(ctx context.Context, pathID uuid.UUID) error

	// Input Nilai & Kalkulasi
	InputScore(ctx context.Context, appID uuid.UUID, componentCode string, scoreVal float64, note string, actorUserID uuid.UUID, actorRoles []string) (*model.Score, *model.ScoreSummary, error)
	GetParticipantScores(ctx context.Context, appID uuid.UUID) ([]model.Score, *model.ScoreSummary, error)

	// Rekapitulasi & Ekspor
	GetProgramRecap(ctx context.Context, periodID, pathID uuid.UUID, programCode string, userRoles []string, userScopes []string) (*model.ProgramRecapResponse, error)
	ExportProgramRecapXLSX(ctx context.Context, periodID, pathID uuid.UUID, programCode string, userRoles []string, userScopes []string) ([]byte, error)

	// Penguncian & Internal API
	SetProgramLock(ctx context.Context, periodID, pathID uuid.UUID, programCode string, isLocked bool, actorUserID *uuid.UUID) (*model.ProgramLock, error)
	GetProgramLock(ctx context.Context, periodID, pathID uuid.UUID, programCode string) (*model.ProgramLock, error)
	GetSummariesInternal(ctx context.Context, periodID uuid.UUID, programCode *string) ([]model.ScoreSummary, error)
}

type scoringService struct {
	cfg          *config.Config
	compRepo     repository.ComponentRepository
	scoreRepo    repository.ScoreRepository
	excelExport  exporter.ExcelExporter
	admissionCli internalclient.Client
	examCli      internalclient.Client
	identityCli  internalclient.Client
}

func NewScoringService(
	cfg *config.Config,
	compRepo repository.ComponentRepository,
	scoreRepo repository.ScoreRepository,
	excelExport exporter.ExcelExporter,
	admissionCli internalclient.Client,
	examCli internalclient.Client,
	identityCli internalclient.Client,
) ScoringService {
	return &scoringService{
		cfg:          cfg,
		compRepo:     compRepo,
		scoreRepo:    scoreRepo,
		excelExport:  excelExport,
		admissionCli: admissionCli,
		examCli:      examCli,
		identityCli:  identityCli,
	}
}

// -------------------------------------------------------------
// Komponen Nilai & Validasi Bobot
// -------------------------------------------------------------

func (s *scoringService) CreateComponent(ctx context.Context, comp *model.ScoreComponent) (*model.ScoreComponent, error) {
	if comp.WeightPercent <= 0 || comp.WeightPercent > 100 {
		return nil, ErrInvalidWeightSum
	}
	if comp.MaxScore <= 0 {
		comp.MaxScore = 100.00
	}

	// Cek duplikasi kode pada jalur ini
	existing, err := s.compRepo.FindByPathAndCode(ctx, comp.PathID, comp.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateComponent
	}

	// Cek apakah penambahan bobot ini menyebabkan total melebihi 100%
	currentTotal, err := s.compRepo.GetTotalWeightByPath(ctx, comp.PathID, nil)
	if err != nil {
		return nil, err
	}
	if (currentTotal + comp.WeightPercent) > 100.0001 {
		return nil, ErrWeightExceeds100
	}

	now := time.Now()
	comp.CreatedAt = now
	comp.UpdatedAt = now

	if err := s.compRepo.Create(ctx, comp); err != nil {
		return nil, err
	}
	return comp, nil
}

func (s *scoringService) UpdateComponent(ctx context.Context, comp *model.ScoreComponent) (*model.ScoreComponent, error) {
	existing, err := s.compRepo.FindByID(ctx, comp.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrComponentNotFound
	}

	if comp.WeightPercent <= 0 || comp.WeightPercent > 100 {
		return nil, ErrInvalidWeightSum
	}
	if comp.MaxScore <= 0 {
		comp.MaxScore = 100.00
	}

	// Cek apakah total bobot komponen aktif lain + bobot baru melebihi 100%
	currentOtherTotal, err := s.compRepo.GetTotalWeightByPath(ctx, existing.PathID, &existing.ID)
	if err != nil {
		return nil, err
	}
	if (currentOtherTotal + comp.WeightPercent) > 100.0001 {
		return nil, ErrWeightExceeds100
	}

	existing.Name = comp.Name
	existing.WeightPercent = comp.WeightPercent
	existing.MaxScore = comp.MaxScore
	existing.IsActive = comp.IsActive
	existing.UpdatedAt = time.Now()

	if err := s.compRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *scoringService) DeleteComponent(ctx context.Context, id uuid.UUID) error {
	existing, err := s.compRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrComponentNotFound
	}
	return s.compRepo.Delete(ctx, id)
}

func (s *scoringService) GetComponentsByPathID(ctx context.Context, pathID uuid.UUID) ([]model.ScoreComponent, error) {
	return s.compRepo.FindByPathID(ctx, pathID)
}

func (s *scoringService) ValidatePathWeightTotal(ctx context.Context, pathID uuid.UUID) error {
	total, err := s.compRepo.GetTotalWeightByPath(ctx, pathID, nil)
	if err != nil {
		return err
	}
	if math.Abs(total-100.00) > 0.001 {
		return ErrInvalidWeightSum
	}
	return nil
}

// -------------------------------------------------------------
// Input Nilai & Kalkulasi Nilai Akhir
// -------------------------------------------------------------

// Struct pembantu untuk parsing respons internal inter-service
type admissionAppResponse struct {
	ID                 uuid.UUID `json:"id"`
	PeriodID           uuid.UUID `json:"period_id"`
	PathID             uuid.UUID `json:"path_id"`
	RegistrationNumber string    `json:"registration_number"`
	UserID             uuid.UUID `json:"user_id"`
	Choices            []struct {
		ChoiceNumber     int    `json:"choice_number"`
		StudyProgramCode string `json:"study_program_code"`
	} `json:"choices"`
}

type examParticipantResponse struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	SessionID     uuid.UUID `json:"session_id"`
}

type examSessionDetailResponse struct {
	ID        uuid.UUID `json:"id"`
	Examiners []struct {
		ExaminerUserID uuid.UUID `json:"examiner_user_id"`
	} `json:"examiners"`
}

func (s *scoringService) InputScore(
	ctx context.Context,
	appID uuid.UUID,
	componentCode string,
	scoreVal float64,
	note string,
	actorUserID uuid.UUID,
	actorRoles []string,
) (*model.Score, *model.ScoreSummary, error) {
	// 1. Ambil informasi pendaftaran dari Admission Service
	appData, err := s.fetchApplication(ctx, appID)
	if err != nil {
		return nil, nil, err
	}

	periodID := appData.PeriodID
	pathID := appData.PathID
	programCode := ""
	for _, ch := range appData.Choices {
		if ch.ChoiceNumber == 1 {
			programCode = ch.StudyProgramCode
			break
		}
	}
	if programCode == "" && len(appData.Choices) > 0 {
		programCode = appData.Choices[0].StudyProgramCode
	}

	// 2. Proteksi Penguncian Nilai (SCORES_LOCKED)
	isLocked, err := s.scoreRepo.IsProgramLocked(ctx, periodID, pathID, programCode)
	if err != nil {
		return nil, nil, err
	}
	if isLocked {
		return nil, nil, ErrScoresLocked
	}

	// 3. Validasi Komponen Nilai
	comp, err := s.compRepo.FindByPathAndCode(ctx, pathID, componentCode)
	if err != nil {
		return nil, nil, err
	}
	if comp == nil || !comp.IsActive {
		return nil, nil, ErrComponentNotFound
	}
	if scoreVal < 0 || scoreVal > comp.MaxScore {
		return nil, nil, ErrInvalidScoreRange
	}

	// 4. Verifikasi Otorisasi Dosen Penguji
	// Jika aktor hanya role dosen (bukan admin_pmb atau superadmin), cek penugasan sesi di Exam Service
	isAdmin := hasAnyRole(actorRoles, "admin_pmb", "superadmin")
	if !isAdmin {
		if err := s.verifyExaminerAssignment(ctx, appID, actorUserID); err != nil {
			return nil, nil, err
		}
	}

	// 5. Simpan / Perbarui Nilai Komponen
	now := time.Now()
	scoreRecord := &model.Score{
		ApplicationID: appID,
		ComponentCode: componentCode,
		Score:         scoreVal,
		InputBy:       actorUserID,
		InputAt:       now,
		Note:          note,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.scoreRepo.UpsertScore(ctx, scoreRecord); err != nil {
		return nil, nil, err
	}

	// 6. Kalkulasi Ulang Nilai Akhir Tertimbang
	allScores, err := s.scoreRepo.FindScoresByApplicationID(ctx, appID)
	if err != nil {
		return nil, nil, err
	}

	allComps, err := s.compRepo.FindByPathID(ctx, pathID)
	if err != nil {
		return nil, nil, err
	}

	compMap := make(map[string]model.ScoreComponent)
	for _, c := range allComps {
		if c.IsActive {
			compMap[c.Code] = c
		}
	}

	var finalScore float64
	for _, sc := range allScores {
		if c, exists := compMap[sc.ComponentCode]; exists {
			finalScore += (sc.Score * c.WeightPercent) / 100.0
		}
	}
	// Pembulatan ke 2 tempat desimal
	finalScore = math.Round(finalScore*100) / 100

	summary := &model.ScoreSummary{
		ApplicationID:    appID,
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: programCode,
		FinalScore:       finalScore,
		ComputedAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.scoreRepo.UpsertSummary(ctx, summary); err != nil {
		return nil, nil, err
	}

	// 7. Hitung Ulang Peringkat Dinamis dalam Program Studi
	if err := s.scoreRepo.RecalculateRanksInProgram(ctx, periodID, pathID, programCode); err != nil {
		return nil, nil, err
	}

	// Ambil kembali summary terbaru dengan rank_in_program
	latestSummary, err := s.scoreRepo.FindSummaryByApplicationID(ctx, appID)
	if err != nil {
		return scoreRecord, summary, nil
	}
	return scoreRecord, latestSummary, nil
}

func (s *scoringService) GetParticipantScores(ctx context.Context, appID uuid.UUID) ([]model.Score, *model.ScoreSummary, error) {
	scores, err := s.scoreRepo.FindScoresByApplicationID(ctx, appID)
	if err != nil {
		return nil, nil, err
	}
	summary, err := s.scoreRepo.FindSummaryByApplicationID(ctx, appID)
	if err != nil {
		return nil, nil, err
	}
	return scores, summary, nil
}

// -------------------------------------------------------------
// Rekapitulasi & Ekspor
// -------------------------------------------------------------

func (s *scoringService) GetProgramRecap(
	ctx context.Context,
	periodID, pathID uuid.UUID,
	programCode string,
	userRoles []string,
	userScopes []string,
) (*model.ProgramRecapResponse, error) {
	// 1. Validasi Scope Role Prodi
	isProdiOnly := hasAnyRole(userRoles, "prodi") && !hasAnyRole(userRoles, "admin_pmb", "superadmin", "pimpinan")
	if isProdiOnly {
		scopeAllowed := false
		for _, sc := range userScopes {
			if sc == programCode {
				scopeAllowed = true
				break
			}
		}
		if !scopeAllowed {
			return nil, ErrForbiddenProgramAccess
		}
	}

	// 2. Ambil Status Kunci
	isLocked, _ := s.scoreRepo.IsProgramLocked(ctx, periodID, pathID, programCode)

	// 3. Ambil Daftar Ringkasan Nilai
	summaries, err := s.scoreRepo.ListSummariesByProgram(ctx, periodID, pathID, programCode)
	if err != nil {
		return nil, err
	}

	// 4. Ambil Daftar Komponen Jalur
	comps, err := s.compRepo.FindByPathID(ctx, pathID)
	if err != nil {
		return nil, err
	}

	// 5. Susun Baris Peserta
	var items []model.ParticipantRecapItem
	for _, sm := range summaries {
		// Ambil skor per komponen peserta
		pScores, _ := s.scoreRepo.FindScoresByApplicationID(ctx, sm.ApplicationID)
		scoreMap := make(map[string]float64)
		for _, ps := range pScores {
			scoreMap[ps.ComponentCode] = ps.Score
		}

		var compDetails []model.ComponentScoreDetail
		for _, cp := range comps {
			if cp.IsActive {
				compDetails = append(compDetails, model.ComponentScoreDetail{
					ComponentCode: cp.Code,
					ComponentName: cp.Name,
					WeightPercent: cp.WeightPercent,
					MaxScore:      cp.MaxScore,
					Score:         scoreMap[cp.Code],
				})
			}
		}

		// Cari nomor pendaftaran dan nama peserta dari Admission/Identity
		regNo := sm.ApplicationID.String()[:8]
		fullName := "Calon Mahasiswa"
		appInfo, errApp := s.fetchApplication(ctx, sm.ApplicationID)
		if errApp == nil && appInfo != nil {
			if appInfo.RegistrationNumber != "" {
				regNo = appInfo.RegistrationNumber
			}
			personName, errPers := s.fetchPersonName(ctx, appInfo.UserID)
			if errPers == nil && personName != "" {
				fullName = personName
			}
		}

		items = append(items, model.ParticipantRecapItem{
			ApplicationID:      sm.ApplicationID,
			RegistrationNumber: regNo,
			FullName:           fullName,
			StudyProgramCode:   sm.StudyProgramCode,
			ComponentScores:    compDetails,
			FinalScore:         sm.FinalScore,
			RankInProgram:      sm.RankInProgram,
		})
	}

	return &model.ProgramRecapResponse{
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: programCode,
		IsLocked:         isLocked,
		TotalApplicants:  len(items),
		Participants:     items,
	}, nil
}

func (s *scoringService) ExportProgramRecapXLSX(
	ctx context.Context,
	periodID, pathID uuid.UUID,
	programCode string,
	userRoles []string,
	userScopes []string,
) ([]byte, error) {
	recap, err := s.GetProgramRecap(ctx, periodID, pathID, programCode, userRoles, userScopes)
	if err != nil {
		return nil, err
	}

	comps, err := s.compRepo.FindByPathID(ctx, pathID)
	if err != nil {
		return nil, err
	}
	var activeComps []model.ScoreComponent
	for _, c := range comps {
		if c.IsActive {
			activeComps = append(activeComps, c)
		}
	}

	return s.excelExport.ExportProgramRecap(recap, activeComps)
}

// -------------------------------------------------------------
// Penguncian & Internal API
// -------------------------------------------------------------

func (s *scoringService) SetProgramLock(
	ctx context.Context,
	periodID, pathID uuid.UUID,
	programCode string,
	isLocked bool,
	actorUserID *uuid.UUID,
) (*model.ProgramLock, error) {
	now := time.Now()
	lock := &model.ProgramLock{
		PeriodID:         periodID,
		PathID:           pathID,
		StudyProgramCode: programCode,
		IsLocked:         isLocked,
		LockedBy:         actorUserID,
		LockedAt:         &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if !isLocked {
		lock.LockedBy = nil
		lock.LockedAt = nil
	}

	if err := s.scoreRepo.SetProgramLock(ctx, lock); err != nil {
		return nil, err
	}
	return s.scoreRepo.GetProgramLock(ctx, periodID, pathID, programCode)
}

func (s *scoringService) GetProgramLock(
	ctx context.Context,
	periodID, pathID uuid.UUID,
	programCode string,
) (*model.ProgramLock, error) {
	return s.scoreRepo.GetProgramLock(ctx, periodID, pathID, programCode)
}

func (s *scoringService) GetSummariesInternal(
	ctx context.Context,
	periodID uuid.UUID,
	programCode *string,
) ([]model.ScoreSummary, error) {
	return s.scoreRepo.ListSummariesInternal(ctx, periodID, programCode)
}

// -------------------------------------------------------------
// Helper Internal Inter-Service
// -------------------------------------------------------------

func (s *scoringService) fetchApplication(ctx context.Context, appID uuid.UUID) (*admissionAppResponse, error) {
	if s.admissionCli == nil {
		// Mock data untuk pengujian lokal jika client belum disetel
		return &admissionAppResponse{
			ID:                 appID,
			PeriodID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			PathID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			RegistrationNumber: "REG-APP-001",
			UserID:             uuid.New(),
			Choices: []struct {
				ChoiceNumber     int    `json:"choice_number"`
				StudyProgramCode string `json:"study_program_code"`
			}{
				{ChoiceNumber: 1, StudyProgramCode: "TIF"},
			},
		}, nil
	}

	var appData admissionAppResponse
	if err := s.admissionCli.Get(ctx, fmt.Sprintf("/internal/applications/%s", appID), nil, &appData); err != nil {
		return nil, ErrApplicationNotFound
	}
	return &appData, nil
}

func (s *scoringService) verifyExaminerAssignment(ctx context.Context, appID uuid.UUID, examinerUserID uuid.UUID) error {
	if s.examCli == nil {
		return nil // Abaikan verifikasi jika exam client tidak disetel (mode offline/testing)
	}

	// 1. Ambil data peserta dari Exam Service
	var partData examParticipantResponse
	if err := s.examCli.Get(ctx, fmt.Sprintf("/internal/participants/%s", appID), nil, &partData); err != nil {
		return ErrForbiddenExaminer
	}

	// 2. Ambil data sesi ujian untuk memeriksa daftar penguji
	var sessData examSessionDetailResponse
	if err := s.examCli.Get(ctx, fmt.Sprintf("/internal/sessions/%s", partData.SessionID), nil, &sessData); err != nil {
		return ErrForbiddenExaminer
	}

	// 3. Cocokkan ID dosen
	isAssigned := false
	for _, ex := range sessData.Examiners {
		if ex.ExaminerUserID == examinerUserID {
			isAssigned = true
			break
		}
	}

	if !isAssigned {
		return ErrForbiddenExaminer
	}
	return nil
}

func (s *scoringService) fetchPersonName(ctx context.Context, userID uuid.UUID) (string, error) {
	if s.identityCli == nil {
		return "", nil
	}

	var personData struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	}
	if err := s.identityCli.Get(ctx, fmt.Sprintf("/internal/persons/%s", userID), nil, &personData); err != nil {
		return "", err
	}

	if personData.FullName != "" {
		return personData.FullName, nil
	}
	return personData.Name, nil
}

func hasAnyRole(roles []string, targets ...string) bool {
	for _, r := range roles {
		for _, t := range targets {
			if r == t {
				return true
			}
		}
	}
	return false
}
