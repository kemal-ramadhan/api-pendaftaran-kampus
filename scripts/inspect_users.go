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

	rows, err := dbAuth.Query(`
		SELECT u.id, u.email, u.full_name, COALESCE(u.phone, '-'), u.status, COALESCE(r.code, 'calon_mahasiswa') as role_code
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		ORDER BY role_code ASC, u.email ASC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("=== DAFTAR SEMUA PENGGUNA DI DATABASE PMB_AUTH ===")
	count := 0
	for rows.Next() {
		count++
		var id, email, fullName, phone, status, roleCode string
		if err := rows.Scan(&id, &email, &fullName, &phone, &status, &roleCode); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[%2d] Role: %-16s | Email: %-32s | Nama: %-26s | Telp: %-14s | Status: %s\n", count, roleCode, email, fullName, phone, status)
	}
}
