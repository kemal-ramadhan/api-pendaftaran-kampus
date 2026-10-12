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

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false;")
	if err != nil {
		log.Fatalf("Error querying databases: %v", err)
	}
	defer rows.Close()

	fmt.Println("Databases:")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf(" - %s\n", name)
	}

	tables, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';")
	if err != nil {
		log.Fatalf("Error querying tables: %v", err)
	}
	defer tables.Close()

	fmt.Println("Tables in pmb_auth (public):")
	for tables.Next() {
		var tname string
		tables.Scan(&tname)
		fmt.Printf(" - %s\n", tname)
	}
}
