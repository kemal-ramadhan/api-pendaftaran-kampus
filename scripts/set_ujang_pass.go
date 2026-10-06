package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db, err := sql.Open("postgres", "postgres://pmb_user:pmb_password@127.0.0.1:5432/pmb_auth?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	password := "Password123!"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		log.Fatal(err)
	}

	res, err := db.Exec("UPDATE applications SET status = 'form_open' WHERE user_id = '32d44743-7a7a-4d22-9e65-56f63f6d91e0';
	if err != nil {
		log.Fatal(err)
	}

	affected, _ := res.RowsAffected()
	fmt.Printf("Updated password for ujang@gmail.com. Affected: %d\n", affected)
}
