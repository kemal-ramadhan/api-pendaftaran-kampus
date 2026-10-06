package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dbAuth, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbAuth.Close()

	var ujangID, fullName, email, status string
	err = dbAuth.QueryRow("SELECT id, full_name, email, status FROM users WHERE email = 'ujang@gmail.com'").Scan(&ujangID, &fullName, &email, &status)
	if err != nil {
		fmt.Println("Ujang not found in pmb_auth:", err)
		return
	}
	fmt.Printf("=== USER UJANG DI PMB_AUTH ===\nID: %s\nName: %s\nEmail: %s\nStatus: %s\n\n", ujangID, fullName, email, status)

	dbAdm, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbAdm.Close()

	rows, err := dbAdm.Query("SELECT id, registration_no, user_id, period_id, admission_path_id, status, created_at FROM applications WHERE user_id = $1", ujangID)
	if err != nil {
		fmt.Println("Error query applications:", err)
		return
	}
	defer rows.Close()

	count := 0
	fmt.Println("=== APPLICATIONS FOR UJANG DI pmb_auth ===")
	for rows.Next() {
		count++
		var appId, regNo, uid, status, createdAt string
		var periodId, pathId int
		if err := rows.Scan(&appId, &regNo, &uid, &periodId, &pathId, &status, &createdAt); err != nil {
			fmt.Println("Scan err:", err)
		} else {
			fmt.Printf("App #%d: ID=%s, RegNo=%s, Period=%d, Path=%d, Status=%s, CreatedAt=%s\n", count, appId, regNo, periodId, pathId, status, createdAt)
		}
	}
	if count == 0 {
		fmt.Println("TIDAK ADA DATA APPLICATION UNTUK USER UJANG!")
	}

	// Check all applications in admission db
	allRows, err := dbAdm.Query("SELECT id, registration_no, user_id, status, created_at FROM applications ORDER BY created_at DESC LIMIT 10;")
	if err == nil {
		defer allRows.Close()
		fmt.Println("\n=== SEMUA APPLICATIONS DI pmb_auth (LIMIT 10) ===")
		for allRows.Next() {
			var appId, regNo, uid, status, createdAt string
			_ = allRows.Scan(&appId, &regNo, &uid, &status, &createdAt)
			fmt.Printf("ID=%s, RegNo=%s, UserID=%s, Status=%s, CreatedAt=%s\n", appId, regNo, uid, status, createdAt)
		}
	}
}
