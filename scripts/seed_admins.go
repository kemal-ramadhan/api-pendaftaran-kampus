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
		log.Fatal(err)
	}
	defer db.Close()

	users := []struct {
		Email    string
		FullName string
		RoleCode string
	}{
		{"superadmin@bth.id", "Super Admin", "superadmin"},
		{"keuangan@bth.id", "Keuangan", "keuangan"},
		{"prodi@bth.id", "Prodi", "prodi"},
		{"verifikator@bth.id", "Verifikator", "admin_pmb"},
		{"penguji@bth.id", "Penguji", "dosen"},
		{"pimpinan@bth.id", "Pimpinan", "pimpinan"},
		{"dekan@bth.id", "Dekan Fakultas", "dekan"},
		{"ahmad.fauzi@example.com", "Ahmad Fauzi", ""},
		{"ujang@gmail.com", "Ujang Mahasiswa", ""},
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range users {
		var roleID string
		if u.RoleCode != "" {
			err := db.QueryRow("SELECT id FROM roles WHERE code = $1", u.RoleCode).Scan(&roleID)
			if err != nil {
				fmt.Printf("Role %s not found: %v\n", u.RoleCode, err)
				continue
			}
		}

		// Check if user exists
		var userID string
		err = db.QueryRow("SELECT id FROM users WHERE email = $1", u.Email).Scan(&userID)
		if err == sql.ErrNoRows {
			// Create user
			userID = uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO users (id, email, full_name, password_hash, status, email_verified_at, created_at, updated_at) 
				VALUES ($1, $2, $3, $4, 'active', $5, $5, $5)`,
				userID, u.Email, u.FullName, string(hashedPassword), time.Now())
			if err != nil {
				fmt.Printf("Failed to create user %s: %v\n", u.Email, err)
				continue
			}
			fmt.Printf("Created user %s\n", u.Email)
		} else if err != nil {
			fmt.Printf("Error checking user %s: %v\n", u.Email, err)
			continue
		} else {
			fmt.Printf("User %s already exists\n", u.Email)
		}

		// Check if user has role
		if roleID != "" {
			var userRoleID string // Not an ID, just check if exists
			err = db.QueryRow("SELECT user_id FROM user_roles WHERE user_id = $1 AND role_id = $2", userID, roleID).Scan(&userRoleID)
			if err == sql.ErrNoRows {
				// Assign role
				_, err = db.Exec(`
					INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, created_at) 
					VALUES ($1, $2, '', '', $3)`,
					userID, roleID, time.Now())
				if err != nil {
					fmt.Printf("Failed to assign role %s to user %s: %v\n", u.RoleCode, u.Email, err)
				} else {
					fmt.Printf("Assigned role %s to user %s\n", u.RoleCode, u.Email)
				}
			} else if err != nil {
				fmt.Printf("Error checking role for user %s: %v\n", u.Email, err)
			} else {
				fmt.Printf("User %s already has role %s\n", u.Email, u.RoleCode)
			}
		}
	}
}
