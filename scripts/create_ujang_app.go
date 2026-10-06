package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ujangID := "32d44743-7a7a-4d22-9e65-56f63f6d91e0"

	var existingID string
	err = db.QueryRow("SELECT id FROM applications WHERE user_id = $1 AND status NOT IN ('cancelled', 'failed', 'expired', 'withdrawn')", ujangID).Scan(&existingID)
	if err == nil {
		fmt.Printf("User Ujang sudah memiliki aplikasi aktif: %s\n", existingID)
		return
	}

	appID := uuid.New().String()
	regNo := "PMB-2026-1-00005"
	now := time.Now()
	choice1 := "00000000-0000-0000-0000-000000000001"
	choice2 := "00000000-0000-0000-0000-000000000002"

	query := `
		INSERT INTO applications (
			id, registration_no, user_id, period_id, admission_path_id,
			choice1_id, choice2_id, status, path_change_count_after_payment,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, 1, 1,
			$4, $5, 'paid', 0,
			$6, $7
		)
	`
	_, err = db.Exec(query, appID, regNo, ujangID, choice1, choice2, now, now)
	if err != nil {
		log.Fatalf("Gagal insert application untuk Ujang: %v", err)
	}

	fmt.Printf("✅ BERHASIL membuat data aplikasi resmi untuk Ujang!\nAppID: %s\nRegNo: %s\nStatus: paid\nUserID: %s\n", appID, regNo, ujangID)
}
