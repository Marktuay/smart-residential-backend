package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"smart-residential-backend/internal/db"
)

// RegistrarUbicacion inserta o actualiza la ubicación de un usuario (guardia)
func RegistrarUbicacion(ctx context.Context, residencialID string, usuarioID int64, latitud, longitud float64) error {
	// Iniciar transacción ya que vamos a insertar en dos tablas
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Eliminar ubicación anterior activa del usuario
	_, _ = tx.ExecContext(ctx, "DELETE FROM rastreo_ubicacion WHERE usuario_id = $1", usuarioID)

	queryActual := `
	INSERT INTO rastreo_ubicacion (residencial_id, usuario_id, latitud, longitud, actualizado_en) 
	VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
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

// EliminarUbicacionActiva remueve la posición activa del guardia al apagar GPS o cerrar sesión
func EliminarUbicacionActiva(ctx context.Context, usuarioID int64) error {
	_, err := db.DB.ExecContext(ctx, "DELETE FROM rastreo_ubicacion WHERE usuario_id = $1", usuarioID)
	return err
}

// ObtenerUbicacionesActivas devuelve las ubicaciones de guardias que han transmitido en los últimos 15 minutos
func ObtenerUbicacionesActivas(ctx context.Context, residencialID string) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if residencialID == "" || residencialID == "GLOBAL" || residencialID == "TODOS" {
		query = `
		SELECT r.usuario_id, COALESCE(u.nombre_completo, u.email), r.latitud, r.longitud, r.actualizado_en, r.residencial_id
		FROM rastreo_ubicacion r
		JOIN usuarios u ON r.usuario_id = u.id
		WHERE r.actualizado_en >= (CURRENT_TIMESTAMP - INTERVAL '15 minutes')
		ORDER BY r.actualizado_en DESC
		`
	} else {
		query = `
		SELECT r.usuario_id, COALESCE(u.nombre_completo, u.email), r.latitud, r.longitud, r.actualizado_en, r.residencial_id
		FROM rastreo_ubicacion r
		JOIN usuarios u ON r.usuario_id = u.id
		WHERE r.residencial_id = $1 AND r.actualizado_en >= (CURRENT_TIMESTAMP - INTERVAL '15 minutes')
		ORDER BY r.actualizado_en DESC
		`
		args = append(args, residencialID)
	}

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ubicaciones []map[string]interface{}
	for rows.Next() {
		var uid int64
		var nombreOEmail string
		var lat, lng float64
		var actualizadoEn string
		var resID string

		if err := rows.Scan(&uid, &nombreOEmail, &lat, &lng, &actualizadoEn, &resID); err != nil {
			return nil, err
		}
		
		ubicaciones = append(ubicaciones, map[string]interface{}{
			"usuario_id":     uid,
			"email":          nombreOEmail,
			"latitud":        lat,
			"longitud":       lng,
			"actualizado_en": actualizadoEn,
			"residencial_id": resID,
		})
	}

	return ubicaciones, nil
}

// ParseFechaFiltro analiza fechas en formatos RFC3339 o locales (asumiendo UTC-6 / America/Managua si no tienen zona horaria)
func ParseFechaFiltro(str string) (time.Time, error) {
	str = strings.TrimSpace(str)
	if str == "" {
		return time.Time{}, errors.New("fecha vacía")
	}

	// 1. Si contiene offset (+ / -) o Z al final, parsear como RFC3339 / ISO
	layoutsWithTZ := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04-07:00",
	}
	for _, layout := range layoutsWithTZ {
		if t, err := time.Parse(layout, str); err == nil {
			return t.UTC(), nil
		}
	}

	// 2. Si no especifica zona horaria (ej: "2026-11-04 15:00:00" o "2026-11-04T15:00:00"),
	// asumir la hora local de Nicaragua (America/Managua, UTC-6) y convertir a UTC
	loc, err := time.LoadLocation("America/Managua")
	if err != nil {
		loc = time.FixedZone("CST", -6*3600)
	}

	layoutsLocal := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layoutsLocal {
		if t, err := time.ParseInLocation(layout, str, loc); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("formato de fecha no reconocido: %s", str)
}

// ObtenerHistorialUsuario obtiene el historial de ubicaciones de un guardia específico o de todos (si usuarioID es 0)
func ObtenerHistorialUsuario(ctx context.Context, residencialID string, usuarioID int64, fechaInicio, fechaFin string) ([]map[string]interface{}, error) {
	tInicio, errInicio := ParseFechaFiltro(fechaInicio)
	tFin, errFin := ParseFechaFiltro(fechaFin)

	var query string
	var rows *sql.Rows
	var err error

	if errInicio == nil && errFin == nil {
		query = `
		SELECT latitud, longitud, registrado_en 
		FROM historial_ubicaciones 
		WHERE (residencial_id = $1 OR $1 = 'GLOBAL' OR $1 = '' OR $1 = 'TODOS') 
		  AND ($2 = 0 OR usuario_id = $2)
		  AND registrado_en >= $3 AND registrado_en <= $4
		ORDER BY registrado_en ASC
		`
		rows, err = db.DB.QueryContext(ctx, query, residencialID, usuarioID, tInicio, tFin)
	} else {
		query = `
		SELECT latitud, longitud, registrado_en 
		FROM historial_ubicaciones 
		WHERE (residencial_id = $1 OR $1 = 'GLOBAL' OR $1 = '' OR $1 = 'TODOS') 
		  AND ($2 = 0 OR usuario_id = $2)
		  AND registrado_en >= $3::TIMESTAMP AND registrado_en <= $4::TIMESTAMP
		ORDER BY registrado_en ASC
		`
		rows, err = db.DB.QueryContext(ctx, query, residencialID, usuarioID, fechaInicio, fechaFin)
	}

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
