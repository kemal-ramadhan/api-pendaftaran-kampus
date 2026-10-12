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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Password123!"), 10)
	if err != nil {
		log.Fatalf("Gagal hash password: %v", err)
	}

	now := time.Now()

	users := []struct {
		id       string
		email    string
		fullName string
		phone    string
	}{
		{
			id:       "b2000001-0000-4000-8000-000000000001",
			email:    "camaba.polos@bth.ac.id",
			fullName: "Siswa Uji Coba Polosan",
			phone:    "081234567890",
		},
		{
			id:       "b2000002-0000-4000-8000-000000000002",
			email:    "user.polos@bth.ac.id",
			fullName: "Calon Mahasiswa Baru (Polosan)",
			phone:    "089876543210",
		},
	}

	for _, u := range users {
		var existingID string
		err = db.QueryRow("SELECT id FROM users WHERE email = $1", u.email).Scan(&existingID)
		userID := u.id
		if err == sql.ErrNoRows {
			_, err = db.Exec(`
				INSERT INTO users (id, email, phone, full_name, password_hash, status, email_verified_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, 'active', $6, $6, $6)
			`, userID, u.email, u.phone, u.fullName, string(hashedPassword), now)
			if err != nil {
				log.Fatalf("Gagal insert user %s: %v", u.email, err)
			}
			fmt.Printf("✅ Akun %s berhasil dibuat (ID: %s)\n", u.email, userID)
		} else if err != nil {
			log.Fatalf("Gagal query user %s: %v", u.email, err)
		} else {
			userID = existingID
			_, err = db.Exec(`
				UPDATE users SET password_hash = $1, status = 'active', email_verified_at = $2, updated_at = $2
				WHERE id = $3
			`, string(hashedPassword), now, userID)
			if err != nil {
				log.Fatalf("Gagal update user %s: %v", u.email, err)
			}
			fmt.Printf("✅ Akun %s berhasil diupdate (ID: %s)\n", u.email, userID)
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
				log.Fatalf("Gagal assign role user %s: %v", u.email, err)
			}
		}
	}

	_ = uuid.Nil
	fmt.Println("🚀 Seeding akun polosan sukses!")
}
