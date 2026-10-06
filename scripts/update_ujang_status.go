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

	ujangID := "32d44743-7a7a-4d22-9e65-56f63f6d91e0"
	res, err := db.Exec("UPDATE applications SET status = 'form_open' WHERE user_id = $1", ujangID)
	if err != nil {
		log.Fatal(err)
	}

	affected, _ := res.RowsAffected()
	fmt.Printf("Updated application status to form_open for Ujang. Rows: %d\n", affected)
}
