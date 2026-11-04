package repository

import (
	"context"
	"fmt"
	"log"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InicializarTablasFlota crea las tablas del módulo de flota en PostgreSQL si no existen
func InicializarTablasFlota(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS flota_vehiculos (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			codigo_unidad VARCHAR(100) NOT NULL,
			tipo_unidad VARCHAR(50) NOT NULL,
			marca_modelo VARCHAR(150),
			placa VARCHAR(50),
			qr_code VARCHAR(150),
			horometro_km_actual NUMERIC DEFAULT 0,
			estado VARCHAR(50) DEFAULT 'OPERATIVO',
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_flota_vehiculos_residencial ON flota_vehiculos(residencial_id);`,

		`CREATE TABLE IF NOT EXISTS solicitudes_flota (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			codigo_solicitud VARCHAR(100) NOT NULL,
			punto_origen_id VARCHAR(150) NOT NULL,
			punto_destino_id VARCHAR(150) NOT NULL,
			vehiculo_id INT,
			conductor_id INT,
			estado_actual VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
			equipo_adicional_json TEXT,
			creado_por INT NOT NULL,
			creado_en TIMESTAMP NOT NULL DEFAULT NOW(),
			actualizado_en TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_solicitudes_flota_residencial ON solicitudes_flota(residencial_id);`,

		`CREATE TABLE IF NOT EXISTS checklist_360_inspecciones (
			id SERIAL PRIMARY KEY,
			solicitud_id INT NOT NULL,
			residencial_id VARCHAR(50) NOT NULL,
			etapa VARCHAR(50) NOT NULL,
			inspector_id INT NOT NULL,
			resultado VARCHAR(50) NOT NULL,
			horometro_km NUMERIC DEFAULT 0,
			nivel_combustible VARCHAR(50),
			observaciones TEXT,
			fecha_inspeccion TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_inspecciones_solicitud ON checklist_360_inspecciones(solicitud_id);`,

		`CREATE TABLE IF NOT EXISTS checklist_items_detalles (
			id SERIAL PRIMARY KEY,
			inspeccion_id INT NOT NULL,
			categoria VARCHAR(50) NOT NULL,
			item_nombre VARCHAR(150) NOT NULL,
			estado_item VARCHAR(50) NOT NULL,
			foto_url TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_items_inspeccion ON checklist_items_detalles(inspeccion_id);`,
	}

	for _, q := range queries {
		if _, err := db.DB.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("error ejecutando migración flota: %v", err)
		}
	}

	log.Println("✅ Tablas e índices de Flota y Checklist 360 verificados exitosamente.")
	return nil
}

// ListarVehiculosFlota retorna los vehículos y maquinaria de un residencial
func ListarVehiculosFlota(ctx context.Context, residencialID string) ([]model.VehiculoFlota, error) {
	query := `
		SELECT id, residencial_id, codigo_unidad, tipo_unidad, marca_modelo, COALESCE(placa, ''), COALESCE(qr_code, ''), horometro_km_actual, estado, created_at
		FROM flota_vehiculos
		WHERE residencial_id = $1
		ORDER BY id DESC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []model.VehiculoFlota
	for rows.Next() {
		var v model.VehiculoFlota
		err := rows.Scan(&v.ID, &v.ResidencialID, &v.CodigoUnidad, &v.TipoUnidad, &v.MarcaModelo, &v.Placa, &v.QrCode, &v.HorometroKmActual, &v.Estado, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		lista = append(lista, v)
	}
	return lista, nil
}

// CrearVehiculoFlota inserta una nueva unidad en la base de datos
func CrearVehiculoFlota(ctx context.Context, v *model.VehiculoFlota) error {
	var count int
	checkQuery := `SELECT count(*) FROM flota_vehiculos WHERE residencial_id = $1 AND (UPPER(codigo_unidad) = UPPER($2) OR ($3 != '' AND UPPER(placa) = UPPER($3)))`
	_ = db.DB.QueryRowContext(ctx, checkQuery, v.ResidencialID, v.CodigoUnidad, v.Placa).Scan(&count)
	if count > 0 {
		return fmt.Errorf("ya existe una unidad registrada con ese código o placa")
	}

	query := `
		INSERT INTO flota_vehiculos (residencial_id, codigo_unidad, tipo_unidad, marca_modelo, placa, qr_code, horometro_km_actual, estado)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return db.DB.QueryRowContext(ctx, query, v.ResidencialID, v.CodigoUnidad, v.TipoUnidad, v.MarcaModelo, v.Placa, v.QrCode, v.HorometroKmActual, v.Estado).Scan(&v.ID, &v.CreatedAt)
}

// ListarSolicitudesFlota retorna las solicitudes de flota con datos relacionales
func ListarSolicitudesFlota(ctx context.Context, residencialID string) ([]model.SolicitudFlota, error) {
	query := `
		SELECT 
			s.id, s.residencial_id, s.codigo_solicitud, s.punto_origen_id, s.punto_destino_id,
			s.vehiculo_id, COALESCE(v.codigo_unidad, ''), COALESCE(v.tipo_unidad, ''),
			s.conductor_id, COALESCE(u.email, ''),
			s.estado_actual, COALESCE(s.equipo_adicional_json, ''),
			s.creado_por, COALESCE(uc.email, ''), s.creado_en, s.actualizado_en
		FROM solicitudes_flota s
		LEFT JOIN flota_vehiculos v ON s.vehiculo_id = v.id
		LEFT JOIN usuarios u ON s.conductor_id = u.id
		LEFT JOIN usuarios uc ON s.creado_por = uc.id
		WHERE s.residencial_id = $1
		ORDER BY s.id DESC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []model.SolicitudFlota
	for rows.Next() {
		var s model.SolicitudFlota
		err := rows.Scan(
			&s.ID, &s.ResidencialID, &s.CodigoSolicitud, &s.PuntoOrigenID, &s.PuntoDestinoID,
			&s.VehiculoID, &s.VehiculoCodigo, &s.VehiculoTipo,
			&s.ConductorID, &s.ConductorNombre,
			&s.EstadoActual, &s.EquipoAdicionalJson,
			&s.CreadoPor, &s.CreadoPorNombre, &s.CreadoEn, &s.ActualizadoEn,
		)
		if err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, nil
}

// CrearSolicitudFlota abre una nueva solicitud de traslado por Operaciones
func CrearSolicitudFlota(ctx context.Context, s *model.SolicitudFlota) error {
	query := `
		INSERT INTO solicitudes_flota (residencial_id, codigo_solicitud, punto_origen_id, punto_destino_id, equipo_adicional_json, estado_actual, creado_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creado_en, actualizado_en
	`
	return db.DB.QueryRowContext(ctx, query,
		s.ResidencialID, s.CodigoSolicitud, s.PuntoOrigenID, s.PuntoDestinoID, s.EquipoAdicionalJson, model.EstadoSolicitudDraft, s.CreadoPor,
	).Scan(&s.ID, &s.CreadoEn, &s.ActualizadoEn)
}

// AsignarSolicitudFlota vincula vehículo y conductor por el área de Transporte
func AsignarSolicitudFlota(ctx context.Context, residencialID string, solicitudID int64, vehiculoID int64, conductorID int64) error {
	query := `
		UPDATE solicitudes_flota
		SET vehiculo_id = $1, conductor_id = $2, estado_actual = $3, actualizado_en = NOW()
		WHERE residencial_id = $4 AND id = $5
	`
	_, err := db.DB.ExecContext(ctx, query, vehiculoID, conductorID, model.EstadoSolicitudAsignada, residencialID, solicitudID)
	return err
}

// ActualizarEstadoSolicitudFlotaFSM realiza la transición de estado FSM
func ActualizarEstadoSolicitudFlotaFSM(ctx context.Context, residencialID string, solicitudID int64, nuevoEstado model.EstadoSolicitudFlota) error {
	query := `
		UPDATE solicitudes_flota
		SET estado_actual = $1, actualizado_en = NOW()
		WHERE residencial_id = $2 AND id = $3
	`
	_, err := db.DB.ExecContext(ctx, query, nuevoEstado, residencialID, solicitudID)
	return err
}

// RegistrarInspeccion360 inserta la inspección y sus detalles en atómico
func RegistrarInspeccion360(ctx context.Context, insp *model.Inspeccion360) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryInsp := `
		INSERT INTO checklist_360_inspecciones (solicitud_id, residencial_id, etapa, inspector_id, resultado, horometro_km, nivel_combustible, observaciones)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, fecha_inspeccion
	`
	err = tx.QueryRowContext(ctx, queryInsp,
		insp.SolicitudID, insp.ResidencialID, insp.Etapa, insp.InspectorID, insp.Resultado, insp.HorometroKm, insp.NivelCombustible, insp.Observaciones,
	).Scan(&insp.ID, &insp.FechaInspeccion)
	if err != nil {
		return err
	}

	queryItem := `
		INSERT INTO checklist_items_detalles (inspeccion_id, categoria, item_nombre, estado_item, foto_url)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, item := range insp.Detalles {
		_, err := tx.ExecContext(ctx, queryItem, insp.ID, item.Categoria, item.ItemNombre, item.EstadoItem, item.FotoUrl)
		if err != nil {
			return err
		}
	}

	// Si la inspección registró horómetro/km nuevo, actualizar el vehículo
	if insp.HorometroKm > 0 {
		updateKm := `
			UPDATE flota_vehiculos 
			SET horometro_km_actual = $1 
			WHERE id = (SELECT vehiculo_id FROM solicitudes_flota WHERE id = $2)
		`
		_, _ = tx.ExecContext(ctx, updateKm, insp.HorometroKm, insp.SolicitudID)
	}

	return tx.Commit()
}

// ObtenerInspeccionesPorSolicitud consulta todas las inspecciones realizadas a una solicitud
func ObtenerInspeccionesPorSolicitud(ctx context.Context, residencialID string, solicitudID int64) ([]model.Inspeccion360, error) {
	query := `
		SELECT i.id, i.solicitud_id, i.residencial_id, i.etapa, i.inspector_id, COALESCE(u.email, ''), i.resultado, i.horometro_km, COALESCE(i.nivel_combustible, ''), COALESCE(i.observaciones, ''), i.fecha_inspeccion
		FROM checklist_360_inspecciones i
		LEFT JOIN usuarios u ON i.inspector_id = u.id
		WHERE i.residencial_id = $1 AND i.solicitud_id = $2
		ORDER BY i.id ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID, solicitudID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []model.Inspeccion360
	for rows.Next() {
		var i model.Inspeccion360
		err := rows.Scan(&i.ID, &i.SolicitudID, &i.ResidencialID, &i.Etapa, &i.InspectorID, &i.InspectorNombre, &i.Resultado, &i.HorometroKm, &i.NivelCombustible, &i.Observaciones, &i.FechaInspeccion)
		if err != nil {
			return nil, err
		}

		// Consultar detalles de ítems
		queryItems := `SELECT id, inspeccion_id, categoria, item_nombre, estado_item, COALESCE(foto_url, '') FROM checklist_items_detalles WHERE inspeccion_id = $1`
		itemRows, err := db.DB.QueryContext(ctx, queryItems, i.ID)
		if err == nil {
			for itemRows.Next() {
				var it model.ItemInspeccion360
				_ = itemRows.Scan(&it.ID, &it.InspeccionID, &it.Categoria, &it.ItemNombre, &it.EstadoItem, &it.FotoUrl)
				i.Detalles = append(i.Detalles, it)
			}
			itemRows.Close()
		}

		lista = append(lista, i)
	}
	return lista, nil
}
