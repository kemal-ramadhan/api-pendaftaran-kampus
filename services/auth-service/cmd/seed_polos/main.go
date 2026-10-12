package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email           string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	Phone           string     `gorm:"type:varchar(20)"`
	PasswordHash    string     `gorm:"type:varchar(255);not null"`
	FullName        string     `gorm:"type:varchar(255);not null"`
	Status          string     `gorm:"type:varchar(20);not null"`
	EmailVerifiedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt       time.Time  `gorm:"type:timestamptz"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz"`
}

func main() {
	dsn := "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Gagal konek DB: %v", err)
	}

	var users []User
	db.Find(&users)
	fmt.Println("=== DAFTAR PENGGUNA TERDAFTAR DI DATABASE PMB_AUTH ===")
	for _, u := range users {
		ver := "Belum"
		if u.EmailVerifiedAt != nil {
			ver = "Sudah"
		}
		fmt.Printf("ID: %s | Email: %s | Nama: %s | Status: %s | Verified: %s\n", u.ID, u.Email, u.FullName, u.Status, ver)
	}

	// Buat hash untuk password123
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	if err != nil {
		log.Fatalf("Gagal hash password: %v", err)
	}

	now := time.Now()

	usersToSeed := []struct {
		Email    string
		FullName string
		Phone    string
	}{
		{"camaba.polos@bth.ac.id", "Calon Mahasiswa Polos", "081299990001"},
		{"user.polos@bth.ac.id", "User Polosan PMB", "081299990002"},
		{"maba.baru@bth.ac.id", "Mahasiswa Baru BTH", "081299990003"},
	}

	for _, item := range usersToSeed {
		var existing User
		res := db.Where("email = ?", item.Email).First(&existing)
		if res.Error == gorm.ErrRecordNotFound {
			newID := uuid.New()
			newUser := User{
				ID:              newID,
				Email:           item.Email,
				FullName:        item.FullName,
				Phone:           item.Phone,
				PasswordHash:    string(hashedPassword),
				Status:          "active",
				EmailVerifiedAt: &now,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := db.Create(&newUser).Error; err != nil {
				log.Fatalf("Gagal buat user %s: %v", item.Email, err)
			}
			fmt.Printf("✅ Berhasil membuat user baru: %s (%s) [password: password123]\n", item.FullName, item.Email)
		} else {
			// Update password_hash, status ke active, dan email_verified_at ke NOW()
			existing.PasswordHash = string(hashedPassword)
			existing.Status = "active"
			existing.EmailVerifiedAt = &now
			existing.UpdatedAt = now
			db.Save(&existing)
			fmt.Printf("✅ Berhasil update user: %s [password: password123]\n", item.Email)
		}
	}
}
