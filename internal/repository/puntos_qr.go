package repository

import (
	"crypto/rand"
	"encoding/hex"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// GenerarCodigoQRSeguro genera un string aleatorio de 16 bytes en formato hex (32 caracteres)
func GenerarCodigoQRSeguro() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback simple si rand falla (muy improbable)
		return "qr-fallback-123456"
	}
	return hex.EncodeToString(bytes)
}

// ListarPuntosQR obtiene todos los puntos QR de un residencial
func ListarPuntosQR(residencialID string) ([]model.PuntoQR, error) {
	query := `
		SELECT p.id, p.residencial_id, p.casa_id, p.nombre, p.codigo_qr, p.latitud, p.longitud, 
		       c.numero_casa, r.nombre AS residente_nombre
		FROM puntos_qr p
		LEFT JOIN casas c ON p.casa_id = c.id
		LEFT JOIN residentes r ON c.id = r.casa_id AND r.es_propietario = true
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
		var numeroCasa *string
		var residenteNombre *string

		if err := rows.Scan(
			&p.ID,
			&p.ResidencialID,
			&casaID,
			&p.Nombre,
			&p.CodigoQR,
			&p.Latitud,
			&p.Longitud,
			&numeroCasa,
			&residenteNombre,
		); err != nil {
			return nil, err
		}

		p.CasaID = casaID
		if numeroCasa != nil {
			p.NumeroCasa = *numeroCasa
		}
		if residenteNombre != nil {
			p.ResidenteNombre = *residenteNombre
		}

		puntos = append(puntos, p)
	}
	return puntos, nil
}

// CrearPuntoQR inserta un nuevo punto QR para el residencial especificado
func CrearPuntoQR(p *model.PuntoQR) error {
	query := `
		INSERT INTO puntos_qr (residencial_id, casa_id, nombre, codigo_qr, latitud, longitud) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id
	`
	
	// Si no se proporcionó un QR explícitamente, generar uno automático
	if p.CodigoQR == "" {
		p.CodigoQR = GenerarCodigoQRSeguro()
	}

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

// EliminarPuntoQR elimina un punto físico (Opcional, útil para mantener limpio)
func EliminarPuntoQR(id int64, residencialID string) error {
	query := `DELETE FROM puntos_qr WHERE id = $1 AND residencial_id = $2`
	_, err := db.DB.Exec(query, id, residencialID)
	return err
}
