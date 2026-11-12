package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq" // Driver nativo para PostgreSQL
)

var DB *sql.DB

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// InicializarDB configura la conexión con PostgreSQL mediante el proxy local
func InicializarDB() error {
	user := getEnvOrDefault("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return fmt.Errorf("la variable de entorno DB_PASSWORD es obligatoria")
	}
	dbname := getEnvOrDefault("DB_NAME", "postgres")
	host := getEnvOrDefault("DB_HOST", "127.0.0.1")
	portStr := getEnvOrDefault("DB_PORT", "5432")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 5432
	}
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

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
