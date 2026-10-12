package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ujangID := "32d44743-7a7a-4d22-9e65-56f63f6d91e0"
	email := "ujang@gmail.com"
	fullName := "Ujang Mahasiswa"
	phone := "08123456789"
	password := "Password123!"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}

	var existingID string
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&existingID)
	if err == sql.ErrNoRows {
		now := time.Now()
		_, err = db.Exec(`
			INSERT INTO users (id, email, phone, full_name, password_hash, status, email_verified_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'active', $6, $6, $6)
		`, ujangID, email, phone, fullName, string(hashedPassword), now)
		if err != nil {
			log.Fatalf("Gagal insert user ujang: %v", err)
		}
		fmt.Printf("✅ User %s (%s) berhasil dibuat dengan password: %s\n", fullName, email, password)
	} else if err != nil {
		log.Fatalf("Gagal cek user: %v", err)
	} else {
		// Update password
		_, err = db.Exec("UPDATE users SET password_hash = $1, status = 'active', email_verified_at = NOW() WHERE id = $2", string(hashedPassword), existingID)
		if err != nil {
			log.Fatalf("Gagal update user: %v", err)
		}
		fmt.Printf("✅ User %s sudah ada, password dan status direset ke aktif (password: %s)\n", email, password)
	}
}
