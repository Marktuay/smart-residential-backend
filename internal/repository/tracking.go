package repository

import (
	"context"
	"smart-residential-backend/internal/db"
)

// RegistrarUbicacion inserta o actualiza la ubicación de un usuario (guardia)
func RegistrarUbicacion(ctx context.Context, residencialID string, usuarioID int64, latitud, longitud float64) error {
	// Iniciar transacción ya que vamos a insertar en dos tablas
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	queryActual := `
	INSERT INTO rastreo_ubicacion (residencial_id, usuario_id, latitud, longitud, actualizado_en) 
	VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	ON CONFLICT (usuario_id) DO UPDATE 
	SET latitud = EXCLUDED.latitud, 
	    longitud = EXCLUDED.longitud, 
	    actualizado_en = CURRENT_TIMESTAMP
	`
	_, err = tx.ExecContext(ctx, queryActual, residencialID, usuarioID, latitud, longitud)
	if err != nil {
		tx.Rollback()
		return err
	}

	queryHistorial := `
	INSERT INTO historial_ubicaciones (residencial_id, usuario_id, latitud, longitud) 
	VALUES ($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, queryHistorial, residencialID, usuarioID, latitud, longitud)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
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

// ObtenerHistorialUsuario obtiene el historial de ubicaciones de un guardia específico en un rango de fechas
func ObtenerHistorialUsuario(ctx context.Context, residencialID string, usuarioID int64, fechaInicio, fechaFin string) ([]map[string]interface{}, error) {
	query := `
	SELECT latitud, longitud, registrado_en 
	FROM historial_ubicaciones 
	WHERE residencial_id = $1 AND usuario_id = $2 
	  AND registrado_en >= $3::TIMESTAMP AND registrado_en <= $4::TIMESTAMP
	ORDER BY registrado_en ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID, usuarioID, fechaInicio, fechaFin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var historial []map[string]interface{}
	for rows.Next() {
		var lat, lng float64
		var registradoEn string
		if err := rows.Scan(&lat, &lng, &registradoEn); err != nil {
			return nil, err
		}
		historial = append(historial, map[string]interface{}{
			"latitud":       lat,
			"longitud":      lng,
			"registrado_en": registradoEn,
		})
	}
	return historial, nil
}

// EliminarUbicacionesAntiguas borra registros del historial más antiguos que los días indicados (Garbage Collector)
func EliminarUbicacionesAntiguas(ctx context.Context, dias int) (int64, error) {
	query := `
	DELETE FROM historial_ubicaciones 
	WHERE registrado_en < NOW() - ($1 || ' days')::INTERVAL
	`
	res, err := db.DB.ExecContext(ctx, query, dias)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
