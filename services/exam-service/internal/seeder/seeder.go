package seeder

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"gorm.io/gorm"
)

func SeedExamSessions(ctx context.Context, db *gorm.DB) error {
	sessionID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	loc := "Gedung Ujian Terpadu PMB Lt. 2 R.201 / CBT Online"

	var existingSession model.ExamSession
	err := db.WithContext(ctx).First(&existingSession, "id = ?", sessionID).Error
	if err != nil {
		session := model.ExamSession{
			ID:       sessionID,
			PeriodID: uuid.MustParse("b0000000-0000-0000-0000-000000000001"),
			PathID:   uuid.MustParse("c0000000-0000-0000-0000-000000000001"),
			Type:     model.SessionTypeWritten,
			Title:    "Ujian Tulis Kemampuan Dasar & Skolastik",
			StartAt:  time.Now().Add(-24 * time.Hour),
			EndAt:    time.Now().Add(60 * 24 * time.Hour),
			Location: &loc,
			Capacity: 200,
			IsActive: true,
		}
		if err := db.WithContext(ctx).Create(&session).Error; err != nil {
			return err
		}
	} else {
		// Pastikan rentang waktu aktif untuk pengerjaan CBT
		db.WithContext(ctx).Model(&existingSession).Updates(map[string]interface{}{
			"start_at":  time.Now().Add(-24 * time.Hour),
			"end_at":    time.Now().Add(60 * 24 * time.Hour),
			"is_active": true,
		})
	}

	// Seed Bank Soal jika belum ada
	var bankCount int64
	db.WithContext(ctx).Model(&model.QuestionBank{}).Count(&bankCount)
	if bankCount == 0 {
		bank1 := model.QuestionBank{
			ID:       uuid.New(),
			Name:     "Matematika Terapan & Farmasi",
			Subject:  "Matematika",
			IsActive: true,
		}
		bank2 := model.QuestionBank{
			ID:       uuid.New(),
			Name:     "Biologi Sel & Farmasi Dasar",
			Subject:  "Biologi",
			IsActive: true,
		}
		bank3 := model.QuestionBank{
			ID:       uuid.New(),
			Name:     "Bahasa Inggris Akademik & Medis",
			Subject:  "Bahasa Inggris",
			IsActive: true,
		}
		bank4 := model.QuestionBank{
			ID:       uuid.New(),
			Name:     "Penalaran Logika & Skolastik",
			Subject:  "TPA & Logika",
			IsActive: true,
		}
		bank5 := model.QuestionBank{
			ID:       uuid.New(),
			Name:     "Etika Profesi Tenaga Kesehatan",
			Subject:  "Etika Kesehatan",
			IsActive: true,
		}

		db.WithContext(ctx).Create(&bank1)
		db.WithContext(ctx).Create(&bank2)
		db.WithContext(ctx).Create(&bank3)
		db.WithContext(ctx).Create(&bank4)
		db.WithContext(ctx).Create(&bank5)

		// Seed Soal Bank 1
		expl1 := "Rumus pengenceran: V1 x C1 = V2 x C2. V1 x 96% = 500 x 70% => V1 = 35000 / 96 = 364.58 mL."
		q1 := model.Question{
			ID:          uuid.New(),
			BankID:      bank1.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Seorang apoteker diminta membuat larutan alkohol 70% sebanyak 500 mL dari sediaan alkohol 96%. Berapakah volume alkohol 96% yang dibutuhkan untuk pengenceran tersebut?",
			Points:      15,
			Difficulty:  "medium",
			Explanation: &expl1,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "364.58 mL", IsCorrect: true},
				{ID: uuid.New(), Label: "B", Content: "350.00 mL", IsCorrect: false},
				{ID: uuid.New(), Label: "C", Content: "420.25 mL", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "280.00 mL", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "385.50 mL", IsCorrect: false},
			},
		}
		expl1b := "Total dosis/hari = 25 mg x 12 kg = 300 mg/hari. Dibagi 3 dosis = 300 / 3 = 100 mg per kali minum."
		q1b := model.Question{
			ID:          uuid.New(),
			BankID:      bank1.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Dosis obat amoksisilin untuk anak adalah 25 mg/kgBB/hari yang dibagi dalam 3 dosis sama rata. Jika berat badan pasien anak 12 kg, berapakah dosis yang diberikan untuk satu kali minum?",
			Points:      15,
			Difficulty:  "easy",
			Explanation: &expl1b,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "100 mg", IsCorrect: true},
				{ID: uuid.New(), Label: "B", Content: "150 mg", IsCorrect: false},
				{ID: uuid.New(), Label: "C", Content: "200 mg", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "300 mg", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "75 mg", IsCorrect: false},
			},
		}
		db.WithContext(ctx).Create(&q1)
		db.WithContext(ctx).Create(&q1b)

		// Seed Soal Bank 2
		expl2 := "Retikulum Endoplasma Kasar (Rough ER) memiliki partikel ribosom yang bertindak sebagai tempat utama sintesis protein sekresi."
		q2 := model.Question{
			ID:          uuid.New(),
			BankID:      bank2.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Organel sel yang berfungsi utama dalam proses sintesis protein dan memiliki partikel ribosom pada permukaan membrannya adalah...",
			Points:      15,
			Difficulty:  "easy",
			Explanation: &expl2,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "Mitokondria", IsCorrect: false},
				{ID: uuid.New(), Label: "B", Content: "Retikulum Endoplasma Kasar", IsCorrect: true},
				{ID: uuid.New(), Label: "C", Content: "Badan Golgi", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "Lisosom", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "Peroksisom", IsCorrect: false},
			},
		}
		expl2b := "Osmosis merupakan difusi pelarut melintasi membran semipermeabel dari potensial air tinggi ke potensial air rendah (hipotonis ke hipertonis)."
		q2b := model.Question{
			ID:          uuid.New(),
			BankID:      bank2.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Proses transpor pasif molekul zat pelarut (seperti air) melintasi membran semipermeabel dari larutan berkonsentrasi rendah ke konsentrasi tinggi disebut...",
			Points:      15,
			Difficulty:  "medium",
			Explanation: &expl2b,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "Difusi sederhana", IsCorrect: false},
				{ID: uuid.New(), Label: "B", Content: "Osmosis", IsCorrect: true},
				{ID: uuid.New(), Label: "C", Content: "Endositosis", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "Eksositosis", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "Transpor aktif primer", IsCorrect: false},
			},
		}
		db.WithContext(ctx).Create(&q2)
		db.WithContext(ctx).Create(&q2b)

		// Seed Soal Bank 3
		expl3 := "The primary global health concern of excessive antibiotic consumption is the selection of drug-resistant bacterial strains."
		q3 := model.Question{
			ID:          uuid.New(),
			BankID:      bank3.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Antibiotics are medications used to treat bacterial infections. They work by killing bacteria or preventing them from reproducing. However, overuse can lead to antibiotic resistance. What is the main concern regarding antibiotic overuse?",
			Points:      15,
			Difficulty:  "medium",
			Explanation: &expl3,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "It cures infections too quickly", IsCorrect: false},
				{ID: uuid.New(), Label: "B", Content: "Bacteria can become resistant to the drugs", IsCorrect: true},
				{ID: uuid.New(), Label: "C", Content: "The cost of medicine decreases", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "Patients become immune to all diseases", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "Antibiotics will lose their color", IsCorrect: false},
			},
		}
		db.WithContext(ctx).Create(&q3)

		// Seed Soal Bank 4
		expl4 := "Silogisme kategori: Karena semua mahasiswa farmasi praktikum kimia dan sebagian memakai kacamata pelindung, maka sebagian mahasiswa mengenakan kacamata pelindung."
		q4 := model.Question{
			ID:          uuid.New(),
			BankID:      bank4.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Semua mahasiswa Fakultas Farmasi BTH wajib mengikuti praktikum kimia dasar. Sebagian peserta praktikum kimia dasar mengenakan kacamata pelindung. Kesimpulan yang paling tepat adalah...",
			Points:      15,
			Difficulty:  "medium",
			Explanation: &expl4,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "Sebagian mahasiswa Fakultas Farmasi BTH mengenakan kacamata pelindung", IsCorrect: true},
				{ID: uuid.New(), Label: "B", Content: "Semua yang memakai kacamata pelindung adalah dosen farmasi", IsCorrect: false},
				{ID: uuid.New(), Label: "C", Content: "Tidak ada mahasiswa yang mengikuti praktikum kimia", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "Semua mahasiswa farmasi tidak pernah praktikum", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "Kacamata pelindung hanya dipakai di luar ruangan", IsCorrect: false},
			},
		}
		db.WithContext(ctx).Create(&q4)

		// Seed Soal Bank 5
		expl5 := "Beneficence adalah prinsip moral yang mewajibkan tenaga medis/kesehatan bertindak demi kebaikan, manfaat maksimal, dan perlindungan pasien."
		q5 := model.Question{
			ID:          uuid.New(),
			BankID:      bank5.ID,
			Type:        model.QuestionTypeSingleChoice,
			Content:     "Prinsip etika profesi tenaga kesehatan yang menekankan kewajiban untuk senantiasa bertindak demi kebaikan, manfaat, dan keselamatan pasien disebut prinsip...",
			Points:      15,
			Difficulty:  "easy",
			Explanation: &expl5,
			IsActive:    true,
			Options: []model.QuestionOption{
				{ID: uuid.New(), Label: "A", Content: "Autonomy", IsCorrect: false},
				{ID: uuid.New(), Label: "B", Content: "Beneficence", IsCorrect: true},
				{ID: uuid.New(), Label: "C", Content: "Non-maleficence", IsCorrect: false},
				{ID: uuid.New(), Label: "D", Content: "Justice", IsCorrect: false},
				{ID: uuid.New(), Label: "E", Content: "Confidentiality", IsCorrect: false},
			},
		}
		db.WithContext(ctx).Create(&q5)

		// Seed CBTExam jika belum ada
		var examCount int64
		db.WithContext(ctx).Model(&model.CBTExam{}).Count(&examCount)
		if examCount == 0 {
			cbtExam := model.CBTExam{
				ID:                      uuid.New(),
				SessionID:               sessionID,
				Title:                   "Ujian CBT Seleksi Mandiri PMB TA 2026/2027",
				Instructions:            "Bacalah setiap butir soal dengan saksama. Pilihlah salah satu jawaban yang paling tepat. Waktu ujian akan berjalan mundur secara otomatis dari server.",
				DurationMinutes:         60,
				QuestionCount:           7,
				RandomizeQuestions:      false,
				RandomizeOptions:        false,
				ShowResultToParticipant: true,
				RequiresAccessToken:     false,
				Status:                  model.CBTExamStatusPublished,
				Rules: []model.CBTExamRule{
					{BankID: bank1.ID, Count: 2},
					{BankID: bank2.ID, Count: 2},
					{BankID: bank3.ID, Count: 1},
					{BankID: bank4.ID, Count: 1},
					{BankID: bank5.ID, Count: 1},
				},
			}
			db.WithContext(ctx).Create(&cbtExam)
		}
	}

	return nil
}
