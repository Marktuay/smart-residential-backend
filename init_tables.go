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

	queries := []string{
		`CREATE TABLE IF NOT EXISTS historial_ubicaciones (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			usuario_id INT NOT NULL,
			latitud DOUBLE PRECISION NOT NULL,
			longitud DOUBLE PRECISION NOT NULL,
			registrado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS puntos_qr (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			nombre VARCHAR(150) NOT NULL,
			codigo_qr VARCHAR(150) NOT NULL,
			latitud NUMERIC,
			longitud NUMERIC
		);`,
		`CREATE TABLE IF NOT EXISTS programas_ronda (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			nombre VARCHAR(150) NOT NULL,
			hora_inicio TIME NOT NULL,
			hora_fin TIME NOT NULL,
			recurrencia VARCHAR(50) NOT NULL,
			puntos INT[]
		);`,
		`CREATE TABLE IF NOT EXISTS asignaciones_ronda (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			programa_id INT REFERENCES programas_ronda(id),
			usuario_id INT,
			fecha DATE NOT NULL,
			estado VARCHAR(50) DEFAULT 'PENDIENTE'
		);`,
		`CREATE TABLE IF NOT EXISTS visitas (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			nombre_visitante VARCHAR(150) NOT NULL,
			cedula_visitante VARCHAR(50) NOT NULL,
			estado VARCHAR(50) DEFAULT 'PENDIENTE',
			fecha_esperada TIMESTAMP NOT NULL,
			numero_casa VARCHAR(50),
			medio_ingreso VARCHAR(50),
			placa_vehiculo VARCHAR(50),
			fecha_ingreso TIMESTAMP,
			fecha_salida TIMESTAMP
		);`,
	}

	for _, q := range queries {
		_, err := db.DB.Exec(q)
		if err != nil {
			log.Printf("Error ejecutando query: %v", err)
		}
	}

	// Añadir placa_vehiculo a visitas si no existe
	_, err := db.DB.Exec(`ALTER TABLE visitas ADD COLUMN IF NOT EXISTS placa_vehiculo VARCHAR(50);`)
	if err != nil {
		log.Printf("Aviso (placa_vehiculo): %v", err)
	}

	// Añadir contrato de seguridad a casas
	_, err = db.DB.Exec(`ALTER TABLE casas ADD COLUMN IF NOT EXISTS tiene_contrato_seguridad BOOLEAN DEFAULT FALSE;`)
	if err != nil {
		log.Printf("Aviso (tiene_contrato_seguridad): %v", err)
	}

	// Añadir casa_id a puntos_qr
	_, err = db.DB.Exec(`ALTER TABLE puntos_qr ADD COLUMN IF NOT EXISTS casa_id INT REFERENCES casas(id) ON DELETE SET NULL;`)
	if err != nil {
		log.Printf("Aviso (casa_id): %v", err)
	}

	// Añadir coordenadas y zoom al residencial
	queriesResidenciales := []string{
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS latitud_centro DOUBLE PRECISION;`,
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS longitud_centro DOUBLE PRECISION;`,
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS zoom_defecto INT DEFAULT 15;`,
	}
	for _, q := range queriesResidenciales {
		_, err := db.DB.Exec(q)
		if err != nil {
			log.Printf("Aviso (residenciales): %v", err)
		}
	}

	fmt.Println("Migraciones ejecutadas exitosamente.")
}
