package repository

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitPuntosQRDB inicializa la tabla de puntos QR si no existe
func InitPuntosQRDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS puntos_qr (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			casa_id BIGINT NULL,
			nombre VARCHAR(150) NOT NULL,
			codigo_qr VARCHAR(100) NOT NULL,
			latitud DOUBLE PRECISION DEFAULT 0.0,
			longitud DOUBLE PRECISION DEFAULT 0.0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_puntos_qr_residencial ON puntos_qr(residencial_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_puntos_qr_codigo ON puntos_qr(residencial_id, codigo_qr);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_puntos_qr_nombre ON puntos_qr(residencial_id, LOWER(TRIM(nombre)));`,
	}

	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error al inicializar tabla puntos_qr: %w", err)
		}
	}
	return nil
}

// GenerarCodigoQRSeguro genera un string aleatorio de 16 bytes en formato hex (32 caracteres)
func GenerarCodigoQRSeguro() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("ncs-%d", db.DB.Stats().OpenConnections)
	}
	return hex.EncodeToString(bytes)
}

// ListarPuntosQR obtiene todos los puntos QR de un residencial con datos de casa y última marcación
func ListarPuntosQR(residencialID string) ([]model.PuntoQR, error) {
	query := `
		SELECT p.id, p.residencial_id, p.casa_id, p.nombre, p.codigo_qr, p.latitud, p.longitud, 
		       COALESCE(c.numero_casa, '') AS numero_casa,
		       COALESCE(c.bloque, '') AS bloque,
		       COALESCE(
		           (SELECT res.nombre 
		            FROM residente_casas rc 
		            JOIN residentes res ON rc.residente_id = res.id 
		            WHERE rc.casa_id = p.casa_id AND res.deleted_at IS NULL 
		            ORDER BY rc.es_principal DESC LIMIT 1),
		           r.nombre, 
		           ''
		       ) AS residente_nombre,
		       COALESCE(c.tiene_contrato_seguridad, false) AS tiene_contrato_seguridad,
		       COALESCE(TO_CHAR((SELECT MAX(fecha_hora) FROM marcaciones_rondas m WHERE m.punto_qr_id = p.id), 'YYYY-MM-DD HH24:MI'), '') AS ultima_marcacion,
		       COALESCE((SELECT COUNT(*) FROM marcaciones_rondas m WHERE m.punto_qr_id = p.id), 0) AS total_marcaciones,
		       TO_CHAR(p.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM puntos_qr p
		LEFT JOIN casas c ON p.casa_id = c.id AND c.deleted_at IS NULL
		LEFT JOIN residentes r ON c.id = r.casa_id AND r.deleted_at IS NULL
		WHERE p.residencial_id = $1
		ORDER BY p.id ASC
	`
	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var puntos []model.PuntoQR
	for rows.Next() {
		var p model.PuntoQR
		var casaID *int64

		if err := rows.Scan(
			&p.ID,
			&p.ResidencialID,
			&casaID,
			&p.Nombre,
			&p.CodigoQR,
			&p.Latitud,
			&p.Longitud,
			&p.NumeroCasa,
			&p.Bloque,
			&p.ResidenteNombre,
			&p.TieneContratoSeguridad,
			&p.UltimaMarcacion,
			&p.TotalMarcaciones,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}

		p.CasaID = casaID
		puntos = append(puntos, p)
	}

	if puntos == nil {
		puntos = []model.PuntoQR{}
	}
	return puntos, nil
}

// CrearPuntoQR inserta un nuevo punto QR con validación anti-duplicados
func CrearPuntoQR(p *model.PuntoQR) error {
	p.Nombre = strings.TrimSpace(p.Nombre)
	if p.Nombre == "" {
		return fmt.Errorf("el nombre del punto QR es obligatorio")
	}

	// Verificar si ya existe un punto con el mismo nombre en este residencial
	var count int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM puntos_qr WHERE residencial_id = $1 AND LOWER(TRIM(nombre)) = LOWER($2)`, p.ResidencialID, p.Nombre).Scan(&count)
	if count > 0 {
		return fmt.Errorf("DUPLICATE_PUNTO_QR: Ya existe un punto de control con el nombre '%s' en este residencial", p.Nombre)
	}

	if p.CodigoQR == "" {
		p.CodigoQR = GenerarCodigoQRSeguro()
	}

	query := `
		INSERT INTO puntos_qr (residencial_id, casa_id, nombre, codigo_qr, latitud, longitud) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id
	`

	err := db.DB.QueryRow(
		query, 
		p.ResidencialID, 
		p.CasaID,
		p.Nombre, 
		p.CodigoQR, 
		p.Latitud, 
		p.Longitud,
	).Scan(&p.ID)

	return err
}

// ActualizarPuntoQR actualiza nombre, ubicación y casa vinculada
func ActualizarPuntoQR(p *model.PuntoQR) error {
	p.Nombre = strings.TrimSpace(p.Nombre)
	if p.Nombre == "" {
		return fmt.Errorf("el nombre del punto QR es obligatorio")
	}

	// Verificar duplicado excluyendo el actual
	var count int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM puntos_qr WHERE residencial_id = $1 AND LOWER(TRIM(nombre)) = LOWER($2) AND id <> $3`, p.ResidencialID, p.Nombre, p.ID).Scan(&count)
	if count > 0 {
		return fmt.Errorf("DUPLICATE_PUNTO_QR: Ya existe otro punto de control con el nombre '%s'", p.Nombre)
	}

	query := `
		UPDATE puntos_qr
		SET nombre = $1, casa_id = $2, latitud = $3, longitud = $4
		WHERE id = $5 AND residencial_id = $6
	`
	res, err := db.DB.Exec(query, p.Nombre, p.CasaID, p.Latitud, p.Longitud, p.ID, p.ResidencialID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("punto QR no encontrado")
	}
	return nil
}

// EliminarPuntoQR elimina un punto de control
func EliminarPuntoQR(id int64, residencialID string) error {
	query := `DELETE FROM puntos_qr WHERE id = $1 AND residencial_id = $2`
	res, err := db.DB.Exec(query, id, residencialID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("punto QR no encontrado o ya eliminado")
	}
	return nil
}

// ObtenerPuntoPorCodigoQR busca un punto por su token QR
func ObtenerPuntoPorCodigoQR(residencialID string, codigoQR string) (*model.PuntoQR, error) {
	query := `
		SELECT p.id, p.residencial_id, p.casa_id, p.nombre, p.codigo_qr, p.latitud, p.longitud,
		       COALESCE(c.numero_casa, ''), COALESCE(c.tiene_contrato_seguridad, false)
		FROM puntos_qr p
		LEFT JOIN casas c ON p.casa_id = c.id
		WHERE p.residencial_id = $1 AND p.codigo_qr = $2
	`
	var p model.PuntoQR
	var casaID *int64
	err := db.DB.QueryRow(query, residencialID, codigoQR).Scan(
		&p.ID, &p.ResidencialID, &casaID, &p.Nombre, &p.CodigoQR, &p.Latitud, &p.Longitud,
		&p.NumeroCasa, &p.TieneContratoSeguridad,
	)
	if err != nil {
		return nil, err
	}
	p.CasaID = casaID
	return &p, nil
}
