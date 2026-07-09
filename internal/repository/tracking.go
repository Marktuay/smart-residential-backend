package repository

import (
	"context"
	"smart-residential-backend/internal/db"
)

// RegistrarUbicacion inserta o actualiza la ubicación de un usuario (guardia)
func RegistrarUbicacion(ctx context.Context, residencialID string, usuarioID int64, latitud, longitud float64) error {
	query := `
	INSERT INTO rastreo_ubicacion (residencial_id, usuario_id, latitud, longitud, actualizado_en) 
	VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	ON CONFLICT (usuario_id) DO UPDATE 
	SET latitud = EXCLUDED.latitud, 
	    longitud = EXCLUDED.longitud, 
	    actualizado_en = CURRENT_TIMESTAMP
	`
	_, err := db.DB.ExecContext(ctx, query, residencialID, usuarioID, latitud, longitud)
	return err
}

// ObtenerUbicacionesActivas devuelve las últimas ubicaciones de los guardias del residencial
// Retorna un slice de mapas para simplificar el JSON en este ejemplo
func ObtenerUbicacionesActivas(ctx context.Context, residencialID string) ([]map[string]interface{}, error) {
	query := `
	SELECT r.usuario_id, u.email, r.latitud, r.longitud, r.actualizado_en 
	FROM rastreo_ubicacion r
	JOIN usuarios u ON r.usuario_id = u.id
	WHERE r.residencial_id = $1
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ubicaciones []map[string]interface{}
	for rows.Next() {
		var uid int64
		var email string
		var lat, lng float64
		var actualizadoEn string

		if err := rows.Scan(&uid, &email, &lat, &lng, &actualizadoEn); err != nil {
			return nil, err
		}
		
		ubicaciones = append(ubicaciones, map[string]interface{}{
			"usuario_id":     uid,
			"email":          email,
			"latitud":        lat,
			"longitud":       lng,
			"actualizado_en": actualizadoEn,
		})
	}

	return ubicaciones, nil
}
