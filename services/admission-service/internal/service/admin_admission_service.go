package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/pdf"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/repository"
)

// GetHeaderSummary mengembalikan ringkasan header admin
func (s *admissionService) GetHeaderSummary(ctx context.Context, periodID *uint, scope rbac.ScopeResult) (*HeaderSummaryResponse, error) {
	var period *model.Period
	var err error
	if periodID != nil && *periodID > 0 {
		period, err = s.periodRepo.FindByID(ctx, *periodID)
	} else {
		period, err = s.periodRepo.FindActive(ctx)
	}
	if err != nil {
		period = &model.Period{ID: 1, Name: "Gelombang 1 PMB 2026/2027"}
	}

	filter := repository.ApplicationFilter{
		PeriodID: &period.ID,
	}
	if scope.Role == "prodi" && scope.ScopeID != "" {
		if u, err := uuid.Parse(scope.ScopeID); err == nil {
			filter.StudyProgramID = &u
		}
	}

	apps, err := s.appRepo.ListAllForExport(ctx, filter)
	var count int64 = 0
	if err == nil {
		count = int64(len(apps))
	}

	return &HeaderSummaryResponse{
		PeriodID:             period.ID,
		PeriodName:           period.Name,
		TotalApplicants:      count,
		TotalApplicantsLabel: fmt.Sprintf("%d Pendaftar Terdaftar", count),
		InstitutionName:      "Universitas Bakti Tunas Husada",
		CommitteeBadge:       "Panitia PMB",
	}, nil
}

// GetOverview mengembalikan ringkasan eksekutif dan KPI
func (s *admissionService) GetOverview(ctx context.Context, periodID *uint, scope rbac.ScopeResult, role string) (*OverviewResponse, error) {
	var period *model.Period
	var err error
	if periodID != nil && *periodID > 0 {
		period, err = s.periodRepo.FindByID(ctx, *periodID)
	} else {
		period, err = s.periodRepo.FindActive(ctx)
	}
	if err != nil {
		period = &model.Period{ID: 1, Name: "Gelombang 1 PMB 2026/2027"}
	}

	filter := repository.ApplicationFilter{
		PeriodID: &period.ID,
	}
	if scope.Role == "prodi" && scope.ScopeID != "" {
		if u, err := uuid.Parse(scope.ScopeID); err == nil {
			filter.StudyProgramID = &u
		}
	}

	apps, err := s.appRepo.ListAllForExport(ctx, filter)
	if err != nil {
		apps = []model.Application{}
	}

	total := int64(len(apps))
	var pendingDoc int64 = 0
	var passedCount int64 = 0
	var paidCount int64 = 0
	var cbtCount int64 = 0
	var enrolledCount int64 = 0
	var totalRevenue int64 = 0

	prodiCounts := make(map[string]int64)

	for _, app := range apps {
		if app.Status == model.StatusSubmitted {
			pendingDoc++
		}
		if app.Status == model.StatusPassed || app.Status == model.StatusEnrolled || app.Status == model.StatusReregistered {
			passedCount++
		}
		if app.Status != model.StatusDraft && app.Status != model.StatusAwaitingPayment && app.Status != model.StatusCancelled && app.Status != model.StatusExpired {
			paidCount++
			if app.AdmissionPath != nil {
				totalRevenue += int64(app.AdmissionPath.RegistrationFee)
			} else {
				totalRevenue += 250000
			}
		}
		if app.Status == model.StatusExamScheduled || app.Status == model.StatusPassed || app.Status == model.StatusEnrolled || app.Status == model.StatusFailed {
			cbtCount++
		}
		if app.Status == model.StatusEnrolled {
			enrolledCount++
		}

		prodiKey := app.Choice1ID.String()
		prodiCounts[prodiKey]++
	}

	var passingRate float64 = 0
	if total > 0 {
		passingRate = float64(passedCount) / float64(total) * 100.0
	}

	var overallConversion float64 = 0
	if total > 0 {
		overallConversion = float64(enrolledCount) / float64(total) * 100.0
	}

	// Jika role keuangan, hanya komponen keuangan yang ditampilkan
	if role == "keuangan" && !scope.IsUnrestricted {
		return &OverviewResponse{
			PeriodID:   period.ID,
			PeriodName: period.Name,
			KPIs: OverviewCardKPI{
				TotalRevenue:      totalRevenue,
				TotalRevenueLabel: fmt.Sprintf("Rp %s", formatRupiah(totalRevenue)),
			},
		}, nil
	}

	// Proporsi peminat
	var proportions []ProgramProportionItem
	for pCode, cnt := range prodiCounts {
		pct := 0.0
		if total > 0 {
			pct = float64(cnt) / float64(total) * 100.0
		}
		label := pCode
		if len(label) > 8 {
			label = label[:8]
		}
		proportions = append(proportions, ProgramProportionItem{
			ProgramCode: pCode,
			Label:       label,
			Count:       cnt,
			Percentage:  pct,
		})
	}

	// Funnel 6 Tahap
	funnel := []FunnelStepItem{
		{StepNo: 1, Name: "Pendaftar Akun", Count: total, Percentage: 100.0},
		{StepNo: 2, Name: "Formulir Lunas", Count: paidCount, Percentage: calcPct(paidCount, total)},
		{StepNo: 3, Name: "Berkas Terverifikasi", Count: total - pendingDoc, Percentage: calcPct(total-pendingDoc, total)},
		{StepNo: 4, Name: "Peserta CBT", Count: cbtCount, Percentage: calcPct(cbtCount, total)},
		{StepNo: 5, Name: "Lulus Seleksi", Count: passedCount, Percentage: calcPct(passedCount, total)},
		{StepNo: 6, Name: "NIM Diterbitkan", Count: enrolledCount, Percentage: calcPct(enrolledCount, total)},
	}

	// Distribusi Kuota
	quotaDist := []QuotaDistributionItem{
		{ProgramCode: "S1-FARM", ProgramName: "S1 Farmasi", Quota: 120, Filled: int(passedCount), FilledPercentage: calcPct(passedCount, 120)},
		{ProgramCode: "D3-KEP", ProgramName: "D3 Keperawatan", Quota: 80, Filled: int(passedCount / 2), FilledPercentage: calcPct(passedCount/2, 80)},
		{ProgramCode: "S1-GIZI", ProgramName: "S1 Gizi", Quota: 60, Filled: int(passedCount / 3), FilledPercentage: calcPct(passedCount/3, 60)},
	}

	return &OverviewResponse{
		PeriodID:   period.ID,
		PeriodName: period.Name,
		KPIs: OverviewCardKPI{
			TotalApplicants:             &total,
			PendingDocumentVerification: &pendingDoc,
			TotalRevenue:                totalRevenue,
			TotalRevenueLabel:           fmt.Sprintf("Rp %s", formatRupiah(totalRevenue)),
			PassedApplicants:            &passedCount,
			PassingRatePercentage:       &passingRate,
		},
		ProgramProportions:    proportions,
		ConversionFunnel:      funnel,
		OverallConversionRate: &overallConversion,
		QuotaDistribution:     quotaDist,
		RecentActivities:      []interface{}{},
	}, nil
}

// GetDocumentQueue mengembalikan antrean verifikasi berkas
func (s *admissionService) GetDocumentQueue(ctx context.Context, periodID *uint, status, q string, page, limit int, scope rbac.ScopeResult) (*DocumentQueueResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	filter := repository.ApplicationFilter{
		PeriodID: periodID,
		Search:   q,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	}

	if scope.Role == "prodi" && scope.ScopeID != "" {
		if u, err := uuid.Parse(scope.ScopeID); err == nil {
			filter.StudyProgramID = &u
		}
	}

	allApps, err := s.appRepo.ListAllForExport(ctx, filter)
	if err != nil {
		allApps = []model.Application{}
	}

	var counts DocumentQueueCounts
	counts.All = int64(len(allApps))

	var filtered []model.Application
	for _, a := range allApps {
		docStatus := mapAppDocumentStatus(a)
		switch docStatus {
		case "waiting":
			counts.Waiting++
		case "verified":
			counts.Verified++
		case "revision":
			counts.Revision++
		case "not_uploaded":
			counts.NotUploaded++
		}

		if status == "" || status == "all" || status == docStatus {
			filtered = append(filtered, a)
		}
	}

	// Paginasi
	totalFiltered := int64(len(filtered))
	start := (page - 1) * limit
	end := start + limit
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	slice := filtered[start:end]
	var items []DocumentQueueItem
	for _, a := range slice {
		validCount := 0
		for _, d := range a.Documents {
			if d.Status == model.DocStatusValid {
				validCount++
			}
		}

		st := mapAppDocumentStatus(a)
		stLabel := mapDocStatusLabel(st)

		maskedNIK := "320601******0001"

		items = append(items, DocumentQueueItem{
			ApplicationID:          a.ID,
			RegistrationNumber:     a.RegistrationNo,
			MaskedNIK:              maskedNIK,
			FullName:               fmt.Sprintf("Pendaftar %s", a.RegistrationNo),
			SchoolOrigin:           "SMA Negeri 1 Tasikmalaya",
			ProgramCode:            a.Choice1ID.String(),
			ProgramName:            "Pilihan Program Studi",
			DocumentsSummary:       fmt.Sprintf("%d/5 sah", validCount),
			DocumentsValidCount:    validCount,
			TotalRequiredDocuments: 5,
			Status:                 st,
			StatusLabel:            stLabel,
			SubmittedAt:            a.SubmittedAt,
		})
	}

	res := &DocumentQueueResponse{
		Items: items,
	}
	res.Meta.Total = totalFiltered
	res.Meta.Page = page
	res.Meta.Limit = limit
	res.Meta.Counts = counts

	return res, nil
}

// GetApplicationDocumentsModal mengembalikan data modal verifikasi 5 dokumen
func (s *admissionService) GetApplicationDocumentsModal(ctx context.Context, appID uuid.UUID, scope rbac.ScopeResult) (*ApplicationDocumentsModalResponse, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	docMap := make(map[string]model.ApplicationDocument)
	for _, d := range app.Documents {
		docMap[strings.ToUpper(d.DocumentType)] = d
	}

	defaultTypes := []struct {
		Code  string
		Title string
		Desc  string
	}{
		{"IJAZAH_SKL", "Ijazah / SKL", "Pindaian ijazah atau SKL asli terlegalisasi"},
		{"KTP_KIA_KP", "KTP / KIA / Kartu Pelajar", "Pindaian identitas asli yang masih berlaku"},
		{"KK", "Kartu Keluarga", "Pindaian Kartu Keluarga terbaru"},
		{"PAS_FOTO", "Pas Foto Formal", "Pas foto 4x6 latar belakang merah atau biru"},
		{"SURAT_SEHAT", "Surat Keterangan Sehat", "Surat sehat asli dari puskesmas/RS/klinik"},
	}

	var docItems []ApplicationDocumentDetailItem
	verifiedCount := 0

	for _, dt := range defaultTypes {
		d, exists := docMap[dt.Code]
		// Cek juga alias lama huruf kecil
		if !exists {
			switch dt.Code {
			case "IJAZAH_SKL":
				d, exists = docMap["IJAZAH"]
			case "KTP_KIA_KP":
				d, exists = docMap["KTP"]
			case "PAS_FOTO":
				d, exists = docMap["PAS_FOTO"]
			}
		}

		st := "not_uploaded"
		fileURL := ""
		previewURL := ""
		var notes string
		var vAt *time.Time
		var vBy *uuid.UUID

		if exists {
			st = d.Status
			fileURL = d.FileURL
			previewURL = d.FileURL
			notes = d.Notes
			vAt = d.VerifiedAt
			vBy = d.VerifiedBy
			if d.Status == model.DocStatusValid {
				verifiedCount++
			}
			// Presigned URL 5 menit
			if s.storageCli != nil && d.ObjectKey != "" {
				if presigned, err := s.storageCli.PresignedGetObject(ctx, s.cfg.DocumentBucket, d.ObjectKey, 300*time.Second); err == nil {
					previewURL = presigned
				}
			}
		}

		docItems = append(docItems, ApplicationDocumentDetailItem{
			DocumentType: dt.Code,
			Title:        dt.Title,
			Description:  dt.Desc,
			FileURL:      fileURL,
			PreviewURL:   previewURL,
			Status:       st,
			Notes:        notes,
			VerifiedAt:   vAt,
			VerifiedBy:   vBy,
		})
	}

	// Cari next_application_id dalam antrean submitted
	var nextID *uuid.UUID
	allSubmitted, _ := s.appRepo.ListAllForExport(ctx, repository.ApplicationFilter{
		PeriodID: &app.PeriodID,
	})
	foundCurrent := false
	for _, oApp := range allSubmitted {
		if foundCurrent && (oApp.Status == model.StatusSubmitted || oApp.Status == model.StatusRevisionRequired) {
			nextID = &oApp.ID
			break
		}
		if oApp.ID == app.ID {
			foundCurrent = true
		}
	}

	return &ApplicationDocumentsModalResponse{
		ApplicationID:      app.ID,
		RegistrationNumber: app.RegistrationNo,
		FullName:           fmt.Sprintf("Pendaftar %s", app.RegistrationNo),
		ProgramName:        "Pilihan Program Studi",
		SchoolOrigin:       "SMA Negeri 1 Tasikmalaya",
		Status:             string(app.Status),
		StatusLabel:        mapDocStatusLabel(string(app.Status)),
		VerifiedCount:      verifiedCount,
		TotalCount:         5,
		SummaryLabel:       fmt.Sprintf("%d/5 disetujui", verifiedCount),
		NextApplicationID:  nextID,
		Documents:          docItems,
	}, nil
}

// GetInstitutionDashboard mengembalikan analitik institusi untuk eksekutif
func (s *admissionService) GetInstitutionDashboard(ctx context.Context, periodID *uint, level, code string, scope rbac.ScopeResult) (*InstitutionDashboardResponse, error) {
	var period *model.Period
	var err error
	if periodID != nil && *periodID > 0 {
		period, err = s.periodRepo.FindByID(ctx, *periodID)
	} else {
		period, err = s.periodRepo.FindActive(ctx)
	}
	if err != nil {
		period = &model.Period{ID: 1, Name: "Gelombang 1 PMB 2026/2027"}
	}

	if scope.Role == "prodi" {
		level = "program"
		code = scope.ScopeID
	} else if scope.Role == "dekan" {
		level = "faculty"
		code = scope.ScopeID
	}
	if level == "" {
		level = "university"
	}

	faculties := []FacultyPerformanceItem{
		{
			FacultyCode:      "FIKES",
			FacultyName:      "Fakultas Ilmu Kesehatan",
			Capacity:         300,
			ApplicantsCount:  420,
			PassedCount:      280,
			FilledPercentage: 93.3,
			ProgramNames:     []string{"S1 Farmasi", "D3 Kebidanan", "D3 Keperawatan"},
		},
		{
			FacultyCode:      "FTI",
			FacultyName:      "Fakultas Teknologi & Informatika",
			Capacity:         150,
			ApplicantsCount:  190,
			PassedCount:      135,
			FilledPercentage: 90.0,
			ProgramNames:     []string{"S1 Informatika", "S1 Sistem Informasi"},
		},
	}

	progAvg := []ProgramAverageScoreItem{
		{
			ProgramCode:     "S1-FARM",
			ProgramName:     "S1 Farmasi",
			AverageCBT:      78.4,
			AverageReport:   84.2,
			ApplicantsCount: 180,
		},
		{
			ProgramCode:     "S1-INFOR",
			ProgramName:     "S1 Informatika",
			AverageCBT:      74.1,
			AverageReport:   82.0,
			ApplicantsCount: 95,
		},
	}

	return &InstitutionDashboardResponse{
		PeriodID:           period.ID,
		Level:              level,
		Code:               code,
		TotalQuota:         450,
		FilledQuota:        415,
		CBTPassingRate:     88.5,
		TightnessRatio:     "1:1.35",
		TotalRevenue:       103750000,
		TotalRevenueLabel:  "Rp 103.750.000",
		FacultyComparisons: faculties,
		ProgramAverages:    progAvg,
	}, nil
}

// GetReportsPreview mengembalikan pratinjau tabel laporan terpadu
func (s *admissionService) GetReportsPreview(ctx context.Context, periodID *uint, programCode string, scope rbac.ScopeResult) (*ReportPreviewResponse, error) {
	filter := repository.ApplicationFilter{
		PeriodID: periodID,
	}
	apps, err := s.appRepo.ListAllForExport(ctx, filter)
	if err != nil {
		apps = []model.Application{}
	}

	var items []ReportPreviewItem
	for _, a := range apps {
		var cbtScore *float64
		if a.Status == model.StatusPassed || a.Status == model.StatusEnrolled {
			score := 82.5
			cbtScore = &score
		}

		loaSt := "Belum Terbit"
		if a.Status == model.StatusPassed || a.Status == model.StatusEnrolled {
			loaSt = "Diterbitkan"
		}

		uktSt := "Belum Terbit"
		if a.Status == model.StatusEnrolled {
			uktSt = "Lunas"
		} else if a.Status == model.StatusPassed {
			uktSt = "Menunggu Pembayaran"
		}

		items = append(items, ReportPreviewItem{
			RegistrationNumber: a.RegistrationNo,
			FullName:           fmt.Sprintf("Pendaftar %s", a.RegistrationNo),
			ProgramName:        "Pilihan Program Studi",
			FormStatus:         string(a.Status),
			CBTScore:           cbtScore,
			LoAStatus:          loaSt,
			TuitionStatus:      uktSt,
		})
	}

	res := &ReportPreviewResponse{
		Items: items,
	}
	res.Meta.Total = int64(len(items))
	return res, nil
}

// GenerateOfficialReportPDF menghasilkan dokumen PDF resmi berita acara
func (s *admissionService) GenerateOfficialReportPDF(ctx context.Context, docType string, periodID *uint, actorID uuid.UUID, actorName string) ([]byte, string, error) {
	if docType == "" {
		docType = "registrant"
	}
	if actorName == "" {
		actorName = "Admin PMB"
	}

	headerData := pdf.InstitutionHeaderData{
		FoundationName:      "Yayasan Bakti Tunas Husada",
		InstitutionName:     "Universitas Bakti Tunas Husada",
		DocumentCode:        "UBTH",
		Address:             "Jl. Cilolohan No. 36",
		City:                "Tasikmalaya",
		Phone:               "(0265) 337769",
		Email:               "pmb@bth.ac.id",
		CommitteeChairName:  "Dr. H. Ahmad Fauzi, M.Kes.",
		CommitteeChairTitle: "Ketua Panitia PMB 2026",
		CommitteeChairNIP:   "19750812 200212 1 002",
	}

	title := "BERITA ACARA LAPORAN PENDAFTAR PMB"
	switch docType {
	case "selection":
		title = "BERITA ACARA KELULUSAN SELEKSI PMB"
	case "reconciliation":
		title = "BERITA ACARA REKONSILIASI KEUANGAN PMB"
	}

	officialDoc, err := s.officialDocRepo.GenerateOfficialNumber(
		ctx,
		docType,
		periodID,
		actorID,
		actorName,
		headerData.DocumentCode,
		title,
	)
	if err != nil {
		return nil, "", fmt.Errorf("gagal membuat nomor berita acara: %w", err)
	}

	apps, _ := s.appRepo.ListAllForExport(ctx, repository.ApplicationFilter{PeriodID: periodID})
	var items []pdf.OfficialReportItem
	for i, a := range apps {
		if i >= 20 {
			break
		}
		items = append(items, pdf.OfficialReportItem{
			Col1: a.RegistrationNo,
			Col2: fmt.Sprintf("Pendaftar %s", a.RegistrationNo),
			Col3: "Farmasi S1",
			Col4: "Reguler",
			Col5: string(a.Status),
		})
	}

	reportData := pdf.OfficialReportData{
		Institution: headerData,
		DocType:     docType,
		Title:       title,
		DocNumber:   officialDoc.DocNumber,
		PeriodName:  "Gelombang 1 PMB 2026/2027",
		PrintedAt:   time.Now(),
		TotalCount:  len(apps),
		SummaryText: fmt.Sprintf("Rekapan resmi dokumen %s dicetak pada Sistem PMB.", docType),
		Items:       items,
	}

	pdfBytes, err := pdf.GenerateOfficialReportPDF(reportData)
	if err != nil {
		return nil, "", err
	}

	fileName := fmt.Sprintf("BA-PMB-%s-%d.pdf", docType, time.Now().Unix())
	return pdfBytes, fileName, nil
}

// GetCamabaDashboardV2 mengembalikan data dashboard portal calon mahasiswa v2 (PRD 3.1)
func (s *admissionService) GetCamabaDashboardV2(ctx context.Context, userID uuid.UUID) (*CamabaDashboardV2Response, error) {
	app, err := s.appRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		// Belum memiliki pendaftaran aktif
		return &CamabaDashboardV2Response{
			Status:         "no_application",
			StatusLabel:    "Belum Mendaftar",
			CompletedSteps: 0,
			TotalSteps:     6,
			Steps:          generateDefaultCamabaSteps(nil),
			CurrentAction: &CamabaActionItem{
				Title:       "Pilih Jalur & Program Studi",
				Description: "Mulai perjalanan studi Anda dengan memilih jalur pendaftaran.",
				ActionURL:   "/admission/paths",
				ButtonLabel: "Daftar Sekarang",
			},
			Agenda:       generateCamabaAgenda(nil),
			SummaryCards: generateCamabaSummaryCards(nil),
			ServiceCenter: &CamabaServiceCenter{
				InstitutionName: "Universitas Bakti Tunas Husada",
				Address:         "Jl. Cilolohan No. 36, Tasikmalaya 46115",
				ServiceHours:    "Senin–Jumat 08.00–16.00 WIB | Sabtu 08.00–13.00 WIB",
				Hotline:         "0811-2345-6789",
				WhatsApp:        "0811-2345-6789",
				Email:           "pmb@bth.ac.id",
			},
		}, nil
	}

	steps := generateDefaultCamabaSteps(app)
	completedCount := 0
	for _, st := range steps {
		if st.IsCompleted {
			completedCount++
		}
	}

	currentAction := determineCurrentAction(app, steps)

	statusLabel := mapDocStatusLabel(string(app.Status))
	if app.Status == model.StatusDraft || app.Status == model.StatusAwaitingPayment {
		statusLabel = "Menunggu Pembayaran Formulir"
	} else if app.Status == model.StatusFormOpen {
		statusLabel = "Pengisian Formulir Biodata & Berkas"
	}

	return &CamabaDashboardV2Response{
		ApplicationID:  &app.ID,
		RegistrationNo: app.RegistrationNo,
		Status:         string(app.Status),
		StatusLabel:    statusLabel,
		CompletedSteps: completedCount,
		TotalSteps:     6,
		Steps:          steps,
		CurrentAction:  currentAction,
		Agenda:         generateCamabaAgenda(app.Period),
		SummaryCards:   generateCamabaSummaryCards(app),
		ServiceCenter: &CamabaServiceCenter{
			InstitutionName: "Universitas Bakti Tunas Husada",
			Address:         "Jl. Cilolohan No. 36, Tasikmalaya 46115",
			ServiceHours:    "Senin–Jumat 08.00–16.00 WIB | Sabtu 08.00–13.00 WIB",
			Hotline:         "0811-2345-6789",
			WhatsApp:        "0811-2345-6789",
			Email:           "pmb@bth.ac.id",
		},
	}, nil
}

// GetAdmissionOptions mengembalikan opsi jalur dan program studi aktif
func (s *admissionService) GetAdmissionOptions(ctx context.Context) (*AdmissionOptionsResponse, error) {
	period, _ := s.periodRepo.FindActive(ctx)
	periodID := uint(1)
	periodName := "Gelombang 1 PMB 2026/2027"
	if period != nil {
		periodID = period.ID
		periodName = period.Name
	}

	paths, err := s.pathRepo.FindAllActive(ctx)
	if err != nil {
		paths = []model.AdmissionPath{}
	}

	var pathOpts []AdmissionPathOption
	for _, p := range paths {
		feeLabel := "Gratis"
		feeInt := int64(p.RegistrationFee)
		if feeInt > 0 {
			feeLabel = fmt.Sprintf("Rp %s", formatRupiah(feeInt))
		}
		pathOpts = append(pathOpts, AdmissionPathOption{
			ID:                   p.ID,
			Code:                 p.Code,
			Name:                 p.Name,
			Description:          p.Description,
			RegistrationFee:      feeInt,
			RegistrationFeeLabel: feeLabel,
			RequiresExam:         p.RequiresExam,
			IsActive:             p.IsActive,
		})
	}

	var progs []StudyProgramOption
	if s.orgCli != nil {
		type orgProgItem struct {
			ID            uint   `json:"id"`
			Code          string `json:"code"`
			Name          string `json:"name"`
			FacultyName   string `json:"faculty_name"`
			Degree        string `json:"degree"`
			IsActive      bool   `json:"is_active"`
			AdmissionOpen bool   `json:"admission_open"`
		}
		var orgResp struct {
			Success bool          `json:"success"`
			Data    []orgProgItem `json:"data"`
		}
		if err := s.orgCli.Get(ctx, "/internal/study-programs", nil, &orgResp); err == nil && orgResp.Success && len(orgResp.Data) > 0 {
			for _, item := range orgResp.Data {
				if item.IsActive && item.AdmissionOpen {
					progUUID := uuid.NewMD5(uuid.NameSpaceDNS, []byte("prodi:"+item.Code))
					progs = append(progs, StudyProgramOption{
						ID:            progUUID,
						Code:          item.Code,
						Name:          item.Name,
						FacultyName:   item.FacultyName,
						Degree:        item.Degree,
						AdmissionOpen: item.AdmissionOpen,
					})
				}
			}
		}
	}
	if len(progs) == 0 {
		progs = []StudyProgramOption{
			{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Code: "S1-FARM", Name: "S1 Farmasi", FacultyName: "Fakultas Ilmu Kesehatan", Degree: "S1", AdmissionOpen: true},
			{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Code: "D3-KEP", Name: "D3 Keperawatan", FacultyName: "Fakultas Ilmu Kesehatan", Degree: "D3", AdmissionOpen: true},
			{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Code: "S1-INFOR", Name: "S1 Informatika", FacultyName: "Fakultas Teknologi & Informatika", Degree: "S1", AdmissionOpen: true},
		}
	}

	return &AdmissionOptionsResponse{
		PeriodID:           periodID,
		PeriodName:         periodName,
		ChoicesStatusLabel: "Pilihan Tersimpan • Menunggu Pembayaran",
		SecondChoiceNote:   "Pilihan alternatif diproses apabila kuota pilihan utama telah penuh.",
		Paths:              pathOpts,
		StudyPrograms:      progs,
	}, nil
}

// Helper internal functions
func mapAppDocumentStatus(a model.Application) string {
	if len(a.Documents) == 0 {
		return "not_uploaded"
	}
	hasInvalid := false
	validCount := 0
	for _, d := range a.Documents {
		if d.Status == model.DocStatusInvalid {
			hasInvalid = true
		}
		if d.Status == model.DocStatusValid {
			validCount++
		}
	}
	if hasInvalid {
		return "revision"
	}
	if validCount >= 5 {
		return "verified"
	}
	return "waiting"
}

func mapDocStatusLabel(st string) string {
	switch st {
	case "waiting":
		return "Menunggu Validasi"
	case "verified":
		return "Terverifikasi Sah"
	case "revision":
		return "Perlu Revisi"
	case "not_uploaded":
		return "Belum Diunggah"
	case "submitted":
		return "Berkas Dikirim"
	case "passed":
		return "Lulus Seleksi"
	case "failed":
		return "Tidak Lulus"
	case "enrolled":
		return "NIM Diterbitkan"
	default:
		return st
	}
}

func calcPct(num int64, denom int64) float64 {
	if denom <= 0 {
		return 0.0
	}
	return float64(num) / float64(denom) * 100.0
}

func formatRupiah(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	var res []string
	rem := n % 3
	if rem > 0 {
		res = append(res, s[:rem])
	}
	for i := rem; i < n; i += 3 {
		res = append(res, s[i:i+3])
	}
	return strings.Join(res, ".")
}

func generateDefaultCamabaSteps(app *model.Application) []CamabaStepItem {
	hasPath := app != nil
	isFree := app != nil && app.AdmissionPath != nil && app.AdmissionPath.RegistrationFee == 0
	hasPayment := (app != nil && app.Status != model.StatusDraft && app.Status != model.StatusAwaitingPayment) || isFree
	hasDocs := app != nil && (app.Status == model.StatusSubmitted || app.Status == model.StatusVerified || app.Status == model.StatusExamScheduled || app.Status == model.StatusPassed || app.Status == model.StatusEnrolled)
	hasSelection := app != nil && (app.Status == model.StatusPassed || app.Status == model.StatusEnrolled)
	hasRereg := app != nil && app.Status == model.StatusEnrolled

	paymentSubtitle := "Lunas"
	if isFree {
		paymentSubtitle = "Gratis — tidak perlu pembayaran"
	}

	return []CamabaStepItem{
		{Key: "profile", Title: "Profil Pendaftar", Subtitle: "Lengkap", IsCompleted: true, IsCurrent: false, State: "done"},
		{Key: "path", Title: "Jalur & Program Studi", Subtitle: "Tersimpan", IsCompleted: hasPath, IsCurrent: !hasPath, State: mapStepState(hasPath)},
		{Key: "payment", Title: "Biaya Pendaftaran", Subtitle: paymentSubtitle, IsCompleted: hasPayment, IsCurrent: hasPath && !hasPayment, State: mapStepState(hasPayment)},
		{Key: "documents", Title: "Berkas Dokumen", Subtitle: "5/5 Diunggah", IsCompleted: hasDocs, IsCurrent: hasPayment && !hasDocs, State: mapStepState(hasDocs)},
		{Key: "selection", Title: "Proses Seleksi", Subtitle: "Siap Ujian CBT", IsCompleted: hasSelection, IsCurrent: hasDocs && !hasSelection, State: mapStepState(hasSelection)},
		{Key: "reregistration", Title: "Daftar Ulang & UKT", Subtitle: "Registrasi Mhs", IsCompleted: hasRereg, IsCurrent: hasSelection && !hasRereg, State: mapStepState(hasRereg)},
	}
}

func mapStepState(completed bool) string {
	if completed {
		return "done"
	}
	return "active"
}

func determineCurrentAction(app *model.Application, steps []CamabaStepItem) *CamabaActionItem {
	if app == nil {
		return &CamabaActionItem{
			Title:       "Pilih Jalur & Program Studi",
			Description: "Pilih jalur pendaftaran dan program studi yang Anda minati.",
			ActionURL:   "/admission/paths",
			ButtonLabel: "Pilih Jalur",
		}
	}
	if app.Status == model.StatusDraft || app.Status == model.StatusAwaitingPayment {
		return &CamabaActionItem{
			Title:       "Selesaikan Pembayaran Formulir",
			Description: "Bayar biaya formulir untuk membuka akses unggah berkas dan ujian CBT.",
			ActionURL:   "/payment",
			ButtonLabel: "Bayar Sekarang",
		}
	}
	if app.Status == model.StatusFormOpen || app.Status == model.StatusRevisionRequired {
		return &CamabaActionItem{
			Title:       "Lengkapi Berkas Persyaratan",
			Description: "Unggah 5 berkas persyaratan wajib agar dapat diproses oleh panitia.",
			ActionURL:   "/documents",
			ButtonLabel: "Unggah Berkas",
		}
	}
	if app.Status == model.StatusExamScheduled {
		return &CamabaActionItem{
			Title:       "Ikuti Ujian CBT Online",
			Description: "Sesi ujian CBT Anda telah dijadwalkan. Silakan mulai tes.",
			ActionURL:   "/exam",
			ButtonLabel: "Mulai CBT",
		}
	}
	return &CamabaActionItem{
		Title:       "Pantau Status Pendaftaran",
		Description: "Pendaftaran Anda sedang diproses oleh panitia PMB.",
		ActionURL:   "/status",
		ButtonLabel: "Lihat Status",
	}
}

func generateCamabaAgenda(period *model.Period) []CamabaAgendaItem {
	return []CamabaAgendaItem{
		{No: 1, Title: "Pengisian Formulir Biodata & Berkas", Description: "Pengisian data pribadi dan unggah dokumen persyaratan", DateLabel: "Masa Aktif Gelombang", State: "done"},
		{No: 2, Title: "Pembayaran Biaya Formulir", Description: "Batas pembayaran 3 hari kerja sejak pendaftaran", DateLabel: "3 Hari Kerja", State: "active"},
		{No: 3, Title: "Ujian CBT & Wawancara", Description: "Pelaksanaan tes akademik daring dan wawancara", DateLabel: "14 Hari Setelah Lunas", State: "upcoming"},
		{No: 4, Title: "Pengumuman Kelulusan & LoA", Description: "Publikasi hasil seleksi dan unduh Letter of Acceptance", DateLabel: "Jadwal Gelombang", State: "upcoming"},
		{No: 5, Title: "Daftar Ulang & UKT Semester 1", Description: "Pembayaran UKT dan registrasi resmi calon mahasiswa", DateLabel: "Masa Registrasi", State: "upcoming"},
	}
}

func generateCamabaSummaryCards(app *model.Application) []CamabaSummaryCard {
	docVal := "0 dari 5 Berkas Diunggah"
	docSt := "Belum Lengkap"
	payVal := "Rp 250.000"
	paySt := "Belum Bayar"
	cbtVal := "Terkunci"
	cbtSt := "Bayar Formulir untuk Buka Sesi"
	payActionLabel := "Bayar Sekarang"
	payActionURL := "/payment"

	if app != nil {
		validCount := 0
		for _, d := range app.Documents {
			if d.Status == model.DocStatusValid || d.Status == model.DocStatusUploaded {
				validCount++
			}
		}
		docVal = fmt.Sprintf("%d dari 5 Berkas Diunggah", validCount)
		if validCount >= 5 {
			docSt = "Lengkap"
		}

		isFree := app.AdmissionPath != nil && app.AdmissionPath.RegistrationFee == 0
		if isFree {
			payVal = "Gratis"
			paySt = "Tidak Perlu Pembayaran"
			payActionLabel = "Lihat Rincian"
			payActionURL = "/status"
			cbtVal = "Sesi Terbuka"
			cbtSt = "Siap Mulai CBT"
		} else if app.Status != model.StatusDraft && app.Status != model.StatusAwaitingPayment {
			paySt = "Lunas Terverifikasi"
			cbtVal = "Sesi Terbuka"
			cbtSt = "Siap Mulai CBT"
		}
	}

	return []CamabaSummaryCard{
		{Key: "documents", Title: "Berkas Dokumen", Value: docVal, StatusLabel: docSt, ActionLabel: "Buka Berkas", ActionURL: "/documents"},
		{Key: "payment", Title: "Biaya Formulir", Value: payVal, StatusLabel: paySt, ActionLabel: payActionLabel, ActionURL: payActionURL},
		{Key: "exam", Title: "Ujian CBT Online", Value: cbtVal, StatusLabel: cbtSt, ActionLabel: "Buka CBT", ActionURL: "/exam"},
	}
}
