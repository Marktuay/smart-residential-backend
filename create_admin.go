package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		log.Fatal("DB_PASSWORD es obligatorio")
	}
	dsn := fmt.Sprintf("host=127.0.0.1 port=5432 user=postgres password=%s dbname=postgres sslmode=disable", dbPass)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	email := os.Getenv("ADMIN_SEED_EMAIL")
	if email == "" {
		email = "informatica@newcenturyni.com"
	}
	password := os.Getenv("ADMIN_SEED_PASSWORD")
	if password == "" {
		log.Fatal("ADMIN_SEED_PASSWORD es obligatorio")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	query := `INSERT INTO usuarios (residencial_id, email, password_hash, rol) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	_, err = db.Exec(query, "Residencial-Las-Torres", email, string(hash), "ADMIN")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Usuario admin creado:", email)
}
