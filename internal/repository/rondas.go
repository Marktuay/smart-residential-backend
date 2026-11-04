package repository

import (
	"context"
	"fmt"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitRondasDB inicializa tablas de rondas, secuencias y marcaciones
func InitRondasDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS programas_rondas (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			nombre VARCHAR(150) NOT NULL,
			hora_inicio VARCHAR(10) NOT NULL,
			hora_fin VARCHAR(10) NOT NULL,
			recurrencia VARCHAR(50) DEFAULT 'DIARIA',
			tolerancia_mins INT DEFAULT 15,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE programas_rondas ADD COLUMN IF NOT EXISTS tolerancia_mins INT DEFAULT 15;`,
		`CREATE INDEX IF NOT EXISTS idx_programas_rondas_res ON programas_rondas(residencial_id);`,

		`CREATE TABLE IF NOT EXISTS programas_puntos (
			id BIGSERIAL PRIMARY KEY,
			programa_id BIGINT NOT NULL,
			punto_qr_id BIGINT NOT NULL,
			orden INT NOT NULL DEFAULT 1
		);`,
		`CREATE INDEX IF NOT EXISTS idx_programas_puntos_prog ON programas_puntos(programa_id);`,

		`CREATE TABLE IF NOT EXISTS asignaciones_rondas (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			programa_id BIGINT NOT NULL,
			usuario_id BIGINT NULL,
			fecha VARCHAR(20) NOT NULL,
			estado VARCHAR(30) DEFAULT 'PROGRAMADA',
			puntos_completados INT DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE asignaciones_rondas ADD COLUMN IF NOT EXISTS puntos_completados INT DEFAULT 0;`,
		`CREATE INDEX IF NOT EXISTS idx_asig_rondas_res ON asignaciones_rondas(residencial_id);`,

		`CREATE TABLE IF NOT EXISTS marcaciones_rondas (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			asignacion_ronda_id BIGINT NULL,
			punto_qr_id BIGINT NOT NULL,
			guardia_id BIGINT NOT NULL,
			fecha_hora TIMESTAMP NOT NULL DEFAULT NOW(),
			latitud DOUBLE PRECISION DEFAULT 0.0,
			longitud DOUBLE PRECISION DEFAULT 0.0,
			observacion TEXT DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_marcaciones_res ON marcaciones_rondas(residencial_id);`,
		`CREATE INDEX IF NOT EXISTS idx_marcaciones_punto ON marcaciones_rondas(punto_qr_id);`,
	}

	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error al inicializar tablas de rondas: %w", err)
		}
	}
	return nil
}

// ObtenerProgramasRondas devuelve los programas configurados con sus puntos detallados
func ObtenerProgramasRondas(ctx context.Context, residencialID string) ([]model.ProgramaRonda, error) {
	query := `
		SELECT pr.id, pr.residencial_id, pr.nombre, pr.hora_inicio, pr.hora_fin, pr.recurrencia,
		       COALESCE(pr.tolerancia_mins, 15),
		       TO_CHAR(pr.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       COUNT(pp.punto_qr_id) AS total_puntos
		FROM programas_rondas pr
		LEFT JOIN programas_puntos pp ON pr.id = pp.programa_id
		WHERE pr.residencial_id = $1
		GROUP BY pr.id, pr.residencial_id, pr.nombre, pr.hora_inicio, pr.hora_fin, pr.recurrencia, pr.tolerancia_mins, pr.created_at
		ORDER BY pr.id ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programas []model.ProgramaRonda
	for rows.Next() {
		var p model.ProgramaRonda
		if err := rows.Scan(
			&p.ID, &p.ResidencialID, &p.Nombre, &p.HoraInicio, &p.HoraFin, &p.Recurrencia,
			&p.ToleranciaMins, &p.CreatedAt, &p.TotalPuntos,
		); err != nil {
			return nil, err
		}

		// Cargar puntos en orden
		puntos, _ := ObtenerPuntosDePrograma(ctx, p.ID)
		p.PuntosDetalle = puntos
		for _, pt := range puntos {
			p.Puntos = append(p.Puntos, pt.ID)
		}

		programas = append(programas, p)
	}

	if programas == nil {
		programas = []model.ProgramaRonda{}
	}
	return programas, nil
}

// ObtenerPuntosDePrograma obtiene los checkpoints asociados a un programa
func ObtenerPuntosDePrograma(ctx context.Context, programaID int64) ([]model.PuntoQR, error) {
	query := `
		SELECT p.id, p.residencial_id, p.casa_id, p.nombre, p.codigo_qr, p.latitud, p.longitud,
		       COALESCE(c.numero_casa, ''), COALESCE(c.tiene_contrato_seguridad, false)
		FROM programas_puntos pp
		JOIN puntos_qr p ON pp.punto_qr_id = p.id
		LEFT JOIN casas c ON p.casa_id = c.id
		WHERE pp.programa_id = $1
		ORDER BY pp.orden ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, programaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var puntos []model.PuntoQR
	for rows.Next() {
		var p model.PuntoQR
		var casaID *int64
		if err := rows.Scan(
			&p.ID, &p.ResidencialID, &casaID, &p.Nombre, &p.CodigoQR, &p.Latitud, &p.Longitud,
			&p.NumeroCasa, &p.TieneContratoSeguridad,
		); err != nil {
			return nil, err
		}
		p.CasaID = casaID
		puntos = append(puntos, p)
	}
	return puntos, nil
}

// CrearProgramaRonda inserta un nuevo programa de patrullaje y sus puntos asociados
func CrearProgramaRonda(ctx context.Context, p *model.ProgramaRonda) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if p.ToleranciaMins <= 0 {
		p.ToleranciaMins = 15
	}
	if p.Recurrencia == "" {
		p.Recurrencia = "DIARIA"
	}

	query := `INSERT INTO programas_rondas (residencial_id, nombre, hora_inicio, hora_fin, recurrencia, tolerancia_mins) 
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = tx.QueryRowContext(ctx, query, p.ResidencialID, p.Nombre, p.HoraInicio, p.HoraFin, p.Recurrencia, p.ToleranciaMins).Scan(&p.ID)
	if err != nil {
		return err
	}

	if len(p.Puntos) > 0 {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO programas_puntos (programa_id, punto_qr_id, orden) VALUES ($1, $2, $3)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for i, puntoID := range p.Puntos {
			if _, err := stmt.ExecContext(ctx, p.ID, puntoID, i+1); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// EliminarProgramaRonda elimina un programa de patrullaje
func EliminarProgramaRonda(ctx context.Context, id int64, residencialID string) error {
	_, _ = db.DB.ExecContext(ctx, "DELETE FROM programas_puntos WHERE programa_id = $1", id)
	query := "DELETE FROM programas_rondas WHERE id = $1 AND residencial_id = $2"
	res, err := db.DB.ExecContext(ctx, query, id, residencialID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("programa no encontrado")
	}
	return nil
}

// ObtenerAsignacionesRondas devuelve las asignaciones de rondas enriquecidas con métricas
func ObtenerAsignacionesRondas(ctx context.Context, residencialID string) ([]model.AsignacionRonda, error) {
	query := `
		SELECT ar.id, ar.residencial_id, ar.programa_id, COALESCE(ar.usuario_id, 0), ar.fecha, ar.estado,
		       COALESCE(ar.puntos_completados, 0),
		       COALESCE(pr.nombre, 'Ronda Sin Nombre') AS programa_nombre,
		       COALESCE(pr.hora_inicio, '') AS hora_inicio,
		       COALESCE(pr.hora_fin, '') AS hora_fin,
		       COALESCE(u.nombre_completo, u.email, 'Sin Asignar') AS guardia_nombre,
		       COALESCE((SELECT COUNT(*) FROM programas_puntos pp WHERE pp.programa_id = ar.programa_id), 0) AS puntos_total,
		       TO_CHAR(ar.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM asignaciones_rondas ar
		JOIN programas_rondas pr ON ar.programa_id = pr.id
		LEFT JOIN usuarios u ON ar.usuario_id = u.id
		WHERE ar.residencial_id = $1
		ORDER BY ar.fecha DESC, ar.id DESC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var asignaciones []model.AsignacionRonda
	for rows.Next() {
		var a model.AsignacionRonda
		if err := rows.Scan(
			&a.ID, &a.ResidencialID, &a.ProgramaID, &a.UsuarioID, &a.Fecha, &a.Estado,
			&a.PuntosCompletados, &a.ProgramaNombre, &a.HoraInicio, &a.HoraFin,
			&a.GuardiaNombre, &a.PuntosTotal, &a.CreatedAt,
		); err != nil {
			return nil, err
		}

		if a.PuntosTotal > 0 {
			a.PorcentajeCumplimiento = float64(a.PuntosCompletados) / float64(a.PuntosTotal) * 100.0
			if a.PorcentajeCumplimiento > 100 {
				a.PorcentajeCumplimiento = 100
			}
		}

		asignaciones = append(asignaciones, a)
	}

	if asignaciones == nil {
		asignaciones = []model.AsignacionRonda{}
	}
	return asignaciones, nil
}

// CrearAsignacionRonda asigna un programa a un usuario en una fecha
func CrearAsignacionRonda(ctx context.Context, a *model.AsignacionRonda) error {
	query := `INSERT INTO asignaciones_rondas (residencial_id, programa_id, usuario_id, fecha, estado, puntos_completados) 
	          VALUES ($1, $2, $3, $4, $5, 0) RETURNING id`
	if a.Estado == "" {
		a.Estado = "PROGRAMADA"
	}
	return db.DB.QueryRowContext(ctx, query, a.ResidencialID, a.ProgramaID, a.UsuarioID, a.Fecha, a.Estado).Scan(&a.ID)
}

// RegistrarMarcacionRonda registra el escaneo de un QR por parte del guardia
func RegistrarMarcacionRonda(ctx context.Context, m *model.MarcacionRonda) error {
	query := `
		INSERT INTO marcaciones_rondas (residencial_id, asignacion_ronda_id, punto_qr_id, guardia_id, latitud, longitud, observacion)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, fecha_hora
	`
	err := db.DB.QueryRowContext(ctx, query,
		m.ResidencialID, m.AsignacionRondaID, m.PuntoQRID, m.GuardiaID, m.Latitud, m.Longitud, m.Observacion,
	).Scan(&m.ID, &m.FechaHora)
	if err != nil {
		return err
	}

	// Si tiene asignación de ronda vinculada, incrementar contador de completados
	if m.AsignacionRondaID != nil && *m.AsignacionRondaID > 0 {
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE asignaciones_rondas 
			SET puntos_completados = puntos_completados + 1,
			    estado = CASE 
			        WHEN puntos_completados + 1 >= (SELECT COUNT(*) FROM programas_puntos pp WHERE pp.programa_id = asignaciones_rondas.programa_id) THEN 'COMPLETADA'
			        ELSE 'EN_CURSO'
			    END
			WHERE id = $1 AND residencial_id = $2
		`, *m.AsignacionRondaID, m.ResidencialID)
	}

	return nil
}

// ListarMarcacionesRondas lista los escaneos recientes
func ListarMarcacionesRondas(ctx context.Context, residencialID string, limite int) ([]model.MarcacionRonda, error) {
	if limite <= 0 || limite > 100 {
		limite = 20
	}
	query := `
		SELECT m.id, m.residencial_id, m.asignacion_ronda_id, m.punto_qr_id,
		       p.nombre AS punto_nombre, COALESCE(c.numero_casa, '') AS numero_casa,
		       m.guardia_id, COALESCE(u.nombre_completo, u.email, 'Guardia') AS guardia_nombre,
		       m.fecha_hora, m.latitud, m.longitud, m.observacion
		FROM marcaciones_rondas m
		JOIN puntos_qr p ON m.punto_qr_id = p.id
		LEFT JOIN casas c ON p.casa_id = c.id
		LEFT JOIN usuarios u ON m.guardia_id = u.id
		WHERE m.residencial_id = $1
		ORDER BY m.fecha_hora DESC
		LIMIT $2
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var marcaciones []model.MarcacionRonda
	for rows.Next() {
		var mr model.MarcacionRonda
		if err := rows.Scan(
			&mr.ID, &mr.ResidencialID, &mr.AsignacionRondaID, &mr.PuntoQRID,
			&mr.PuntoNombre, &mr.NumeroCasa,
			&mr.GuardiaID, &mr.GuardiaNombre,
			&mr.FechaHora, &mr.Latitud, &mr.Longitud, &mr.Observacion,
		); err != nil {
			return nil, err
		}
		marcaciones = append(marcaciones, mr)
	}
	if marcaciones == nil {
		marcaciones = []model.MarcacionRonda{}
	}
	return marcaciones, nil
}
