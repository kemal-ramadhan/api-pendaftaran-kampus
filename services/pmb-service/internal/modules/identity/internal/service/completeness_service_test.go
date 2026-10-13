package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/identity/internal/model"
)

func TestCompletenessService(t *testing.T) {
	personRepo := newMockPersonRepo()
	compSvc := NewCompletenessService(personRepo)
	ctx := context.Background()

	user := uuid.New()

	// 1. User belum punya record apa pun -> complete: false, missing list lengkap
	res, err := compSvc.CheckCompleteness(ctx, user)
	if err != nil {
		t.Fatalf("CheckCompleteness gagal: %v", err)
	}
	if res.Complete {
		t.Fatalf("User tanpa data harus berstatus complete: false")
	}
	if len(res.Missing) == 0 {
		t.Fatalf("Missing fields tidak boleh kosong")
	}

	// 2. User baru isi sebagian profil (tanpa alamat, wali, pendidikan)
	birthDate := time.Date(2005, 5, 20, 0, 0, 0, 0, time.UTC)
	person := &model.Person{
		UserID:     user,
		FullName:   "Budi Santoso",
		NIK:        "enc-nik",
		NIKHash:    "hash-nik",
		BirthPlace: "Surabaya",
		BirthDate:  &birthDate,
		Gender:     "L",
		Religion:   "Islam",
		Phone:      "081299887766",
	}
	_ = personRepo.Save(ctx, person)

	res, err = compSvc.CheckCompleteness(ctx, user)
	if err != nil {
		t.Fatalf("CheckCompleteness gagal: %v", err)
	}
	if res.Complete {
		t.Fatalf("Profil saja tanpa alamat/wali/pendidikan belum boleh complete")
	}

	// 3. Tambahkan Alamat KTP
	person.Addresses = []model.Address{
		{
			Type:       "ktp",
			Street:     "Jl. Pahlawan No. 10",
			Village:    "Gubeng",
			District:   "Gubeng",
			City:       "Surabaya",
			Province:   "Jawa Timur",
			PostalCode: "60281",
		},
	}
	_ = personRepo.Save(ctx, person)

	res, err = compSvc.CheckCompleteness(ctx, user)
	if res.Complete {
		t.Fatalf("Belum boleh complete tanpa wali dan pendidikan")
	}

	// 4. Tambahkan Wali (Ayah)
	person.Guardians = []model.Guardian{
		{
			Relation: "ayah",
			Name:     "Agus Santoso",
			Phone:    "081233445566",
		},
	}
	_ = personRepo.Save(ctx, person)

	res, err = compSvc.CheckCompleteness(ctx, user)
	if res.Complete {
		t.Fatalf("Belum boleh complete tanpa riwayat pendidikan")
	}

	// 5. Tambahkan Pendidikan Terakhir -> harus menjadi COMPLETE = TRUE!
	person.EducationHistories = []model.EducationHistory{
		{
			Level:          "SMA",
			SchoolName:     "SMA Negeri 5 Surabaya",
			GraduationYear: 2023,
		},
	}
	_ = personRepo.Save(ctx, person)

	res, err = compSvc.CheckCompleteness(ctx, user)
	if err != nil {
		t.Fatalf("CheckCompleteness gagal: %v", err)
	}
	if !res.Complete {
		t.Fatalf("Seluruh data sudah terisi, harus complete: true. Missing: %v", res.Missing)
	}
	if len(res.Missing) != 0 {
		t.Fatalf("Missing list harus kosong, didapat: %v", res.Missing)
	}
}
