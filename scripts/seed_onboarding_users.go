package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type DummyUser struct {
	ID        string
	Email     string
	FullName  string
	Phone     string
	Password  string
	ProdiNote string
}

func main() {
	db, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatalf("Gagal terhubung ke database pmb_auth: %v", err)
	}
	defer db.Close()

	// Pastikan role calon_mahasiswa ada
	var roleID int
	err = db.QueryRow("SELECT id FROM roles WHERE code = 'calon_mahasiswa'").Scan(&roleID)
	if err != nil {
		log.Fatalf("Role calon_mahasiswa tidak ditemukan: %v", err)
	}

	dummyList := []DummyUser{
		{
			ID:        "a1000001-0000-4000-8000-000000000001",
			Email:     "maba1.rizky@gmail.com",
			FullName:  "Muhammad Rizky Pratama",
			Phone:     "081234560001",
			Password:  "Password123!",
			ProdiNote: "S1 Farmasi",
		},
		{
			ID:        "a1000002-0000-4000-8000-000000000002",
			Email:     "maba2.anisa@gmail.com",
			FullName:  "Anisa Nurul Aini",
			Phone:     "081234560002",
			Password:  "Password123!",
			ProdiNote: "S1 Keperawatan",
		},
		{
			ID:        "a1000003-0000-4000-8000-000000000003",
			Email:     "maba3.bagas@gmail.com",
			FullName:  "Bagas Dwi Aditya",
			Phone:     "081234560003",
			Password:  "Password123!",
			ProdiNote: "D3 Analis Kesehatan (TLM)",
		},
		{
			ID:        "a1000004-0000-4000-8000-000000000004",
			Email:     "maba4.cantika@gmail.com",
			FullName:  "Cantika Putri Dewi",
			Phone:     "081234560004",
			Password:  "Password123!",
			ProdiNote: "S1 Kebidanan",
		},
		{
			ID:        "a1000005-0000-4000-8000-000000000005",
			Email:     "maba5.dimas@gmail.com",
			FullName:  "Dimas Arya Nugraha",
			Phone:     "081234560005",
			Password:  "Password123!",
			ProdiNote: "S1 Administrasi Rumah Sakit",
		},
		{
			ID:        "a1000006-0000-4000-8000-000000000006",
			Email:     "maba6.fauzan@gmail.com",
			FullName:  "Ahmad Fauzan Hakim",
			Phone:     "081234560006",
			Password:  "Password123!",
			ProdiNote: "S1 Farmasi (Jalur Kemitraan BK)",
		},
		{
			ID:        "a1000007-0000-4000-8000-000000000007",
			Email:     "maba7.gita@gmail.com",
			FullName:  "Gita Lestari Prameswari",
			Phone:     "081234560007",
			Password:  "Password123!",
			ProdiNote: "D3 Farmasi",
		},
		{
			ID:        "a1000008-0000-4000-8000-000000000008",
			Email:     "maba8.hendra@gmail.com",
			FullName:  "Hendra Setiawan",
			Phone:     "081234560008",
			Password:  "Password123!",
			ProdiNote: "S1 Keperawatan",
		},
		{
			ID:        "a1000009-0000-4000-8000-000000000009",
			Email:     "maba9.intan@gmail.com",
			FullName:  "Intan Permata Sari",
			Phone:     "081234560009",
			Password:  "Password123!",
			ProdiNote: "S1 Kebidanan",
		},
		{
			ID:        "a1000010-0000-4000-8000-000000000010",
			Email:     "maba10.kevin@gmail.com",
			FullName:  "Kevin Sanjaya Pratama",
			Phone:     "081234560010",
			Password:  "Password123!",
			ProdiNote: "S1 Administrasi Rumah Sakit",
		},
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Password123!"), 10)
	if err != nil {
		log.Fatalf("Gagal hash password: %v", err)
	}

	now := time.Now()
	fmt.Printf("🚀 Memulai seeding 10 akun dummy calon mahasiswa baru (status onboarding)...\n")

	for i, u := range dummyList {
		var existingID string
		err := db.QueryRow("SELECT id FROM users WHERE email = $1", u.Email).Scan(&existingID)
		userID := u.ID
		if err == sql.ErrNoRows {
			_, err = db.Exec(`
				INSERT INTO users (id, email, phone, full_name, password_hash, status, email_verified_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, 'active', $6, $6, $6)
			`, userID, u.Email, u.Phone, u.FullName, string(hashedPassword), now)
			if err != nil {
				log.Fatalf("Gagal insert user %s: %v", u.Email, err)
			}
		} else if err != nil {
			log.Fatalf("Gagal query user %s: %v", u.Email, err)
		} else {
			userID = existingID
			_, err = db.Exec(`
				UPDATE users SET password_hash = $1, status = 'active', email_verified_at = $2, updated_at = $2
				WHERE id = $3
			`, string(hashedPassword), now, userID)
			if err != nil {
				log.Fatalf("Gagal update user %s: %v", u.Email, err)
			}
		}

		// Pastikan user_roles terpasang
		var existsRole bool
		_ = db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2)", userID, roleID).Scan(&existsRole)
		if !existsRole {
			_, err = db.Exec(`
				INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, created_at)
				VALUES ($1, $2, '', '', $3)
			`, userID, roleID, now)
			if err != nil {
				log.Fatalf("Gagal assign role user %s: %v", u.Email, err)
			}
		}

		fmt.Printf("[%2d/10] ✅ User: %s | Email: %s | Peminatan: %s (Status: Aktif Baru, Siap Onboarding)\n", i+1, u.FullName, u.Email, u.ProdiNote)
	}

	// Buat UUID helper test
	_ = uuid.Nil
	fmt.Println("\n✨ Selesai! Seluruh 10 akun dummy calon mahasiswa telah tersimpan di PostgreSQL.")
}
