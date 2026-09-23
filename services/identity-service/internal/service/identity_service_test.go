package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/crypto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/identity-service/internal/repository"
)

// Mock Person Repository
type mockPersonRepo struct {
	persons map[uuid.UUID]*model.Person
}

func newMockPersonRepo() *mockPersonRepo {
	return &mockPersonRepo{persons: make(map[uuid.UUID]*model.Person)}
}

func (m *mockPersonRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Person, error) {
	p, ok := m.persons[userID]
	if !ok {
		return nil, repository.ErrPersonNotFound
	}
	return p, nil
}

func (m *mockPersonRepo) FindByNIKHash(ctx context.Context, nikHash string) (*model.Person, error) {
	for _, p := range m.persons {
		if p.NIKHash == nikHash {
			return p, nil
		}
	}
	return nil, repository.ErrPersonNotFound
}

func (m *mockPersonRepo) Save(ctx context.Context, person *model.Person) error {
	for _, p := range m.persons {
		if p.NIKHash == person.NIKHash && p.UserID != person.UserID {
			return repository.ErrNIKDuplicate
		}
	}
	m.persons[person.UserID] = person
	return nil
}

func (m *mockPersonRepo) UpdatePhoto(ctx context.Context, userID uuid.UUID, photoKey string) error {
	p, ok := m.persons[userID]
	if !ok {
		return repository.ErrPersonNotFound
	}
	p.PhotoObjectKey = photoKey
	return nil
}

func (m *mockPersonRepo) GetDetailWithRelations(ctx context.Context, userID uuid.UUID) (*model.Person, error) {
	return m.FindByUserID(ctx, userID)
}

func (m *mockPersonRepo) FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]model.Person, error) {
	var list []model.Person
	for _, uid := range userIDs {
		if p, ok := m.persons[uid]; ok {
			list = append(list, *p)
		}
	}
	return list, nil
}

// Mock Address Repository
type mockAddressRepo struct {
	addrs map[string]*model.Address
}

func newMockAddressRepo() *mockAddressRepo {
	return &mockAddressRepo{addrs: make(map[string]*model.Address)}
}

func (m *mockAddressRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Address, error) {
	var list []model.Address
	for _, a := range m.addrs {
		if a.UserID == userID {
			list = append(list, *a)
		}
	}
	return list, nil
}

func (m *mockAddressRepo) FindByUserIDAndType(ctx context.Context, userID uuid.UUID, addrType string) (*model.Address, error) {
	key := userID.String() + ":" + addrType
	if a, ok := m.addrs[key]; ok {
		return a, nil
	}
	return nil, nil
}

func (m *mockAddressRepo) Save(ctx context.Context, address *model.Address) error {
	if address.ID == uuid.Nil {
		address.ID = uuid.New()
	}
	key := address.UserID.String() + ":" + address.Type
	m.addrs[key] = address
	return nil
}

// Mock Guardian Repository
type mockGuardianRepo struct {
	guardians map[string]*model.Guardian
}

func newMockGuardianRepo() *mockGuardianRepo {
	return &mockGuardianRepo{guardians: make(map[string]*model.Guardian)}
}

func (m *mockGuardianRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Guardian, error) {
	var list []model.Guardian
	for _, g := range m.guardians {
		if g.UserID == userID {
			list = append(list, *g)
		}
	}
	return list, nil
}

func (m *mockGuardianRepo) FindByUserIDAndRelation(ctx context.Context, userID uuid.UUID, relation string) (*model.Guardian, error) {
	key := userID.String() + ":" + relation
	if g, ok := m.guardians[key]; ok {
		return g, nil
	}
	return nil, nil
}

func (m *mockGuardianRepo) Save(ctx context.Context, guardian *model.Guardian) error {
	if guardian.ID == uuid.Nil {
		guardian.ID = uuid.New()
	}
	key := guardian.UserID.String() + ":" + guardian.Relation
	m.guardians[key] = guardian
	return nil
}

// Mock Education Repository
type mockEducationRepo struct {
	edus map[uuid.UUID][]model.EducationHistory
}

func newMockEducationRepo() *mockEducationRepo {
	return &mockEducationRepo{edus: make(map[uuid.UUID][]model.EducationHistory)}
}

func (m *mockEducationRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.EducationHistory, error) {
	return m.edus[userID], nil
}

func (m *mockEducationRepo) Save(ctx context.Context, edu *model.EducationHistory) error {
	if edu.ID == uuid.Nil {
		edu.ID = uuid.New()
	}
	m.edus[edu.UserID] = append(m.edus[edu.UserID], *edu)
	return nil
}

func (m *mockEducationRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	delete(m.edus, userID)
	return nil
}

func TestIdentityService_ProfileFlow(t *testing.T) {
	key := crypto.DeriveKey("test-secret-key-32-bytes-abcdef0123")
	personRepo := newMockPersonRepo()
	addressRepo := newMockAddressRepo()
	guardianRepo := newMockGuardianRepo()
	eduRepo := newMockEducationRepo()

	svc := NewIdentityService(personRepo, addressRepo, guardianRepo, eduRepo, nil, key, "pmb-identity-photos")
	ctx := context.Background()

	userA := uuid.New()
	rawNIK := "3201011234560001"

	// 1. Update Profile User A
	req := dto.ProfileRequest{
		FullName:   "Calon Mahasiswa A",
		NIK:        rawNIK,
		BirthPlace: "Bandung",
		BirthDate:  "2005-08-17",
		Gender:     "L",
		Religion:   "Islam",
		Phone:      "081234567890",
	}

	profile, err := svc.UpdateProfile(ctx, userA, req)
	if err != nil {
		t.Fatalf("UpdateProfile gagal: %v", err)
	}

	if profile.FullName != "Calon Mahasiswa A" || profile.NIK != rawNIK {
		t.Fatalf("Data profil tidak cocok: %+v", profile)
	}

	// 2. User A membaca profilnya sendiri -> NIK tidak dimasking
	readA, err := svc.GetProfile(ctx, userA, userA, []string{"calon_mahasiswa"})
	if err != nil {
		t.Fatalf("User A GetProfile gagal: %v", err)
	}
	if readA.NIK != rawNIK {
		t.Fatalf("NIK pemilik harus unmasked, didapat: %s", readA.NIK)
	}

	// 3. User B membaca profil User A -> NIK wajib dimasking!
	userB := uuid.New()
	readB, err := svc.GetProfile(ctx, userA, userB, []string{"calon_mahasiswa"})
	if err != nil {
		t.Fatalf("User B GetProfile gagal: %v", err)
	}
	expectedMasked := crypto.MaskNIK(rawNIK)
	if readB.NIK != expectedMasked {
		t.Fatalf("NIK untuk selain pemilik/admin harus dimasking: didapat %s, diharapkan %s", readB.NIK, expectedMasked)
	}

	// 4. Admin PMB membaca profil User A -> NIK unmasked
	readAdmin, err := svc.GetProfile(ctx, userA, userB, []string{"admin_pmb"})
	if err != nil {
		t.Fatalf("Admin GetProfile gagal: %v", err)
	}
	if readAdmin.NIK != rawNIK {
		t.Fatalf("NIK untuk admin harus unmasked, didapat: %s", readAdmin.NIK)
	}

	// 5. Cek di storage model: NIK di DB terenkripsi (tidak sama dengan plain)
	stored, err := personRepo.FindByUserID(ctx, userA)
	if err != nil {
		t.Fatalf("Gagal ambil stored person: %v", err)
	}
	if stored.NIK == rawNIK {
		t.Fatalf("CRITICAL: NIK tersimpan plain text di database!")
	}
	if stored.NIKHash != crypto.HashNIK(rawNIK) {
		t.Fatalf("NIKHash tidak sesuai")
	}

	// 6. Test duplikasi NIK oleh User B -> harus ditolak
	reqDuplicate := dto.ProfileRequest{
		FullName:   "Calon Mahasiswa B",
		NIK:        rawNIK, // NIK sama dengan user A!
		BirthPlace: "Jakarta",
		BirthDate:  "2005-01-01",
		Gender:     "P",
		Religion:   "Kristen",
		Phone:      "089876543210",
	}
	_, err = svc.UpdateProfile(ctx, userB, reqDuplicate)
	if err == nil {
		t.Fatalf("Pendaftaran NIK ganda harus gagal!")
	}
	if err != repository.ErrNIKDuplicate {
		t.Fatalf("Error harus ErrNIKDuplicate, didapat: %v", err)
	}
}

func TestIdentityService_AddressGuardianEducation(t *testing.T) {
	key := crypto.DeriveKey("test-secret-key-32-bytes-abcdef0123")
	personRepo := newMockPersonRepo()
	addressRepo := newMockAddressRepo()
	guardianRepo := newMockGuardianRepo()
	eduRepo := newMockEducationRepo()

	svc := NewIdentityService(personRepo, addressRepo, guardianRepo, eduRepo, nil, key, "pmb-identity-photos")
	ctx := context.Background()
	user := uuid.New()

	// 1. Simpan Alamat KTP
	addr, err := svc.SaveAddress(ctx, user, dto.AddressRequest{
		Type:       "ktp",
		Street:     "Jl. Merdeka No. 45",
		RT:         "01",
		RW:         "02",
		Village:    "Babakan",
		District:   "Coblong",
		City:       "Bandung",
		Province:   "Jawa Barat",
		PostalCode: "40132",
	})
	if err != nil {
		t.Fatalf("SaveAddress gagal: %v", err)
	}
	if addr.City != "Bandung" {
		t.Fatalf("Kota tidak sesuai: %s", addr.City)
	}

	// 2. Simpan Orang Tua (Ibu)
	g, err := svc.SaveGuardian(ctx, user, dto.GuardianRequest{
		Relation:    "ibu",
		Name:        "Siti Aminah",
		Occupation:  "Guru",
		IncomeRange: "5jt-10jt",
		Phone:       "081122334455",
	})
	if err != nil {
		t.Fatalf("SaveGuardian gagal: %v", err)
	}
	if g.Relation != "ibu" || g.Name != "Siti Aminah" {
		t.Fatalf("Guardian tidak sesuai: %+v", g)
	}

	// 3. Simpan Riwayat Pendidikan
	edu, err := svc.SaveEducation(ctx, user, dto.EducationHistoryRequest{
		Level:          "SMA",
		SchoolName:     "SMA Negeri 1 Bandung",
		Major:          "IPA",
		GraduationYear: 2024,
	})
	if err != nil {
		t.Fatalf("SaveEducation gagal: %v", err)
	}
	if edu.GraduationYear != 2024 {
		t.Fatalf("Tahun kelulusan tidak sesuai: %d", edu.GraduationYear)
	}
}
