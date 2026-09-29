package seeder

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"gorm.io/gorm"
)

func SeedAdmissionData(ctx context.Context, db *gorm.DB) error {
	// 1. Seed Periode Aktif
	var periodCount int64
	db.WithContext(ctx).Model(&model.Period{}).Count(&periodCount)
	var activePeriod model.Period
	examDate := time.Now().AddDate(0, 1, 0)
	announcementDate := time.Now().AddDate(0, 1, 14)
	reregStart := time.Now().AddDate(0, 1, 15)
	reregEnd := time.Now().AddDate(0, 1, 30)

	if periodCount == 0 {
		activePeriod = model.Period{
			Name:                "Gelombang 1 PMB 2026/2027",
			AcademicYear:        "2026/2027",
			StartDate:           time.Now().AddDate(0, -1, 0), // Mulai 1 bulan lalu
			EndDate:             time.Now().AddDate(0, 3, 0),  // Berakhir 3 bulan ke depan
			IsActive:            true,
			ExamDate:            &examDate,
			AnnouncementDate:    &announcementDate,
			ReregistrationStart: &reregStart,
			ReregistrationEnd:   &reregEnd,
		}
		if err := db.WithContext(ctx).Create(&activePeriod).Error; err != nil {
			return err
		}
		log.Println("[ADMISSION-SEEDER] Berhasil seed periode aktif:", activePeriod.Name)
	} else {
		_ = db.WithContext(ctx).Where("is_active = ?", true).First(&activePeriod).Error
		if activePeriod.ExamDate == nil {
			db.WithContext(ctx).Model(&activePeriod).Updates(map[string]interface{}{
				"exam_date":            examDate,
				"announcement_date":    announcementDate,
				"reregistration_start": reregStart,
				"reregistration_end":   reregEnd,
			})
		}
	}

	// 2. Seed 4 Jalur Masuk
	var pathCount int64
	db.WithContext(ctx).Model(&model.AdmissionPath{}).Count(&pathCount)
	if pathCount == 0 {
		paths := []model.AdmissionPath{
			{
				Name:            "Reguler",
				Code:            "REG",
				Description:     "Jalur seleksi melalui ujian tertulis/CBT online",
				RegistrationFee: 250000,
				RequiresExam:    true,
				IsActive:        true,
			},
			{
				Name:            "Prestasi Akademik",
				Code:            "PRES-AK",
				Description:     "Jalur seleksi bebas tes berbasis nilai rapor atau olimpiade sains",
				RegistrationFee: 200000,
				RequiresExam:    false,
				IsActive:        true,
			},
			{
				Name:            "Prestasi Non-Akademik",
				Code:            "PRES-NON",
				Description:     "Jalur seleksi prestasi seni, olahraga, dan keagamaan (Tahfidz)",
				RegistrationFee: 200000,
				RequiresExam:    false,
				IsActive:        true,
			},
			{
				Name:            "Beasiswa Kemitraan",
				Code:            "BEASISWA",
				Description:     "Jalur beasiswa penuh kerja sama institusi dan mitra",
				RegistrationFee: 0,
				RequiresExam:    true,
				IsActive:        true,
			},
		}

		for _, p := range paths {
			pathItem := p
			if err := db.WithContext(ctx).Create(&pathItem).Error; err != nil {
				return err
			}

			// Tambahkan 5 persyaratan dokumen default (PRD 2.2 & 3.4)
			docs := []model.DocumentRequirement{
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "IJAZAH_SKL",
					Name:            "Ijazah / Surat Keterangan Lulus (SKL)",
					Description:     "Unggah pindaian ijazah atau SKL asli terlegalisasi",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "KTP_KIA_KP",
					Name:            "Kartu Identitas (KTP / KIA / Kartu Pelajar)",
					Description:     "Unggah pindaian KTP, KIA, atau Kartu Pelajar asli",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "KK",
					Name:            "Kartu Keluarga (KK)",
					Description:     "Unggah pindaian Kartu Keluarga terbaru",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "PAS_FOTO",
					Name:            "Pas Foto Formal Berwarna",
					Description:     "Unggah pas foto formal terbaru ukuran 4x6 latar belakang merah atau biru",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "SURAT_SEHAT",
					Name:            "Surat Keterangan Sehat",
					Description:     "Unggah surat keterangan sehat asli dari puskesmas, rumah sakit, atau klinik",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
			}

			if pathItem.Code == "PRES-AK" || pathItem.Code == "PRES-NON" {
				docs = append(docs, model.DocumentRequirement{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "sertifikat_prestasi",
					Name:            "Sertifikat Prestasi / Piagam Kejuaraan",
					Description:     "Scan sertifikat kejuaraan minimal tingkat kota/kabupaten",
					IsRequired:      false,
					MaxSizeMB:       5,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				})
			}

			for _, doc := range docs {
				docItem := doc
				if docItem.ID == uuid.Nil {
					docItem.ID = uuid.New()
				}
				_ = db.WithContext(ctx).Create(&docItem).Error
			}
		}
		log.Println("[ADMISSION-SEEDER] Berhasil seed 4 jalur masuk dan 5 dokumen persyaratan default")
	} else {
		// Pastikan SURAT_SEHAT selalu ada di setiap jalur aktif
		var existingPaths []model.AdmissionPath
		db.WithContext(ctx).Find(&existingPaths)
		for _, ep := range existingPaths {
			var suratSehatCount int64
			db.WithContext(ctx).Model(&model.DocumentRequirement{}).
				Where("admission_path_id = ? AND UPPER(document_type) = ?", ep.ID, "SURAT_SEHAT").
				Count(&suratSehatCount)
			if suratSehatCount == 0 {
				docItem := model.DocumentRequirement{
					ID:              uuid.New(),
					AdmissionPathID: ep.ID,
					DocumentType:    "SURAT_SEHAT",
					Name:            "Surat Keterangan Sehat",
					Description:     "Unggah surat keterangan sehat asli dari puskesmas, rumah sakit, atau klinik",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				}
				_ = db.WithContext(ctx).Create(&docItem).Error
			}
		}
	}

	return nil
}
