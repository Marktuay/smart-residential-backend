package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := "host=127.0.0.1 port=5432 user=postgres password=Mark8721!@#$ dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	password := "admin123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	query := `INSERT INTO usuarios (residencial_id, email, password_hash, rol) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	_, err = db.Exec(query, "Residencial-Las-Torres", "admin@lastorres.com", string(hash), "ADMIN")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Usuario admin creado: admin@lastorres.com / admin123")
}
