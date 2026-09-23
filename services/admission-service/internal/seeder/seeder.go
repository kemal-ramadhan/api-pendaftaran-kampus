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
	if periodCount == 0 {
		activePeriod = model.Period{
			Name:         "Gelombang 1 PMB 2026/2027",
			AcademicYear: "2026/2027",
			StartDate:    time.Now().AddDate(0, -1, 0), // Mulai 1 bulan lalu
			EndDate:      time.Now().AddDate(0, 3, 0),  // Berakhir 3 bulan ke depan
			IsActive:     true,
		}
		if err := db.WithContext(ctx).Create(&activePeriod).Error; err != nil {
			return err
		}
		log.Println("[ADMISSION-SEEDER] Berhasil seed periode aktif:", activePeriod.Name)
	} else {
		_ = db.WithContext(ctx).Where("is_active = ?", true).First(&activePeriod).Error
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

			// Tambahkan persyaratan dokumen dasar untuk setiap jalur
			docs := []model.DocumentRequirement{
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "ijazah",
					Name:            "Ijazah / Surat Keterangan Lulus (SKL)",
					Description:     "Scan dokumen asli Ijazah atau SKL bagi lulusan tahun berjalan",
					IsRequired:      true,
					MaxSizeMB:       5,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "rapor",
					Name:            "Transkrip Nilai / Rapor Semester 1-5",
					Description:     "Scan buku rapor atau transkrip nilai resmi yang telah dilegalisir",
					IsRequired:      true,
					MaxSizeMB:       10,
					AllowedFormats:  "pdf",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "ktp",
					Name:            "Kartu Tanda Penduduk (KTP) / Kartu Pelajar",
					Description:     "Scan KTP asli atau Kartu Pelajar yang masih berlaku",
					IsRequired:      true,
					MaxSizeMB:       3,
					AllowedFormats:  "jpg,jpeg,png,pdf",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "kk",
					Name:            "Kartu Keluarga (KK)",
					Description:     "Scan Kartu Keluarga terbaru",
					IsRequired:      true,
					MaxSizeMB:       3,
					AllowedFormats:  "pdf,jpg,jpeg,png",
				},
				{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "pas_foto",
					Name:            "Pas Foto Formal Berwarna",
					Description:     "Pas foto terbaru berpakaian rapi dengan latar belakang merah/biru",
					IsRequired:      true,
					MaxSizeMB:       2,
					AllowedFormats:  "jpg,jpeg,png",
				},
			}

			if pathItem.Code == "PRES-AK" || pathItem.Code == "PRES-NON" {
				docs = append(docs, model.DocumentRequirement{
					AdmissionPathID: pathItem.ID,
					DocumentType:    "sertifikat_prestasi",
					Name:            "Sertifikat Prestasi / Piagam Kejuaraan",
					Description:     "Scan sertifikat kejuaraan minimal tingkat kota/kabupaten",
					IsRequired:      true,
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
		log.Println("[ADMISSION-SEEDER] Berhasil seed 4 jalur masuk dan dokumen persyaratan")
	}

	return nil
}
