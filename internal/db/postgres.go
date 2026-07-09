package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" // Driver nativo para PostgreSQL
)

var DB *sql.DB

// InicializarDB configura la conexión con PostgreSQL mediante el proxy local
func InicializarDB() error {
	// Parámetros por defecto para conectarnos al proxy local de Cloud SQL
	// que está corriendo en el puerto 5432
	user := "postgres"
	password := "Mark8721!@#$"
	dbname := "postgres"
	host := "127.0.0.1"
	port := 5432

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("error al abrir conexión con la base de datos: %v", err)
	}

	// Configuración eficiente del pool de conexiones para alta concurrencia
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Validar la red y autenticación forzando un Ping
	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("error haciendo ping a la base de datos: %v", err)
	}

	log.Println("✅ Conexión al pool de PostgreSQL (Google Cloud SQL) establecida exitosamente.")
	return nil
}
