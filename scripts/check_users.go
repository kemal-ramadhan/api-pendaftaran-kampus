package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, email, full_name, status, email_verified_at FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("=== USERS IN PMB_AUTH ===")
	for rows.Next() {
		var id, email, fullName, status string
		var verified sql.NullTime
		if err := rows.Scan(&id, &email, &fullName, &status, &verified); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %s | Email: %s | Name: %s | Status: %s | Verified: %v\n", id, email, fullName, status, verified.Valid)
	}
}
