package main

import (
	"fmt"
	"log"
	"smart-residential-backend/internal/db"
)

func main() {
	if err := db.InicializarDB(); err != nil {
		log.Fatalf("Error inicializando db: %v", err)
	}

	queryCasas := `
	CREATE TABLE IF NOT EXISTS casas (
		id SERIAL PRIMARY KEY,
		residencial_id VARCHAR(50) NOT NULL,
		numero_casa VARCHAR(50) NOT NULL,
		bloque VARCHAR(50),
		estado VARCHAR(50) DEFAULT 'OCUPADA',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.DB.Exec(queryCasas)
	if err != nil {
		log.Fatalf("Error creando tabla casas: %v", err)
	}

	queryResidentes := `
	CREATE TABLE IF NOT EXISTS residentes (
		id SERIAL PRIMARY KEY,
		residencial_id VARCHAR(50) NOT NULL,
		casa_id INT REFERENCES casas(id) ON DELETE SET NULL,
		nombre VARCHAR(150) NOT NULL,
		telefono VARCHAR(50),
		email VARCHAR(150),
		es_propietario BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.DB.Exec(queryResidentes)
	if err != nil {
		log.Fatalf("Error creando tabla residentes: %v", err)
	}

	fmt.Println("Tablas 'casas' y 'residentes' creadas exitosamente.")
}
