package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// ObtenerProgramasRondas devuelve los programas configurados para un residencial
func ObtenerProgramasRondas(ctx context.Context, residencialID string) ([]model.ProgramaRonda, error) {
	rows, err := db.DB.QueryContext(ctx, "SELECT id, residencial_id, nombre, hora_inicio, hora_fin, recurrencia FROM programas_rondas WHERE residencial_id = $1", residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programas []model.ProgramaRonda
	for rows.Next() {
		var p model.ProgramaRonda
		if err := rows.Scan(&p.ID, &p.ResidencialID, &p.Nombre, &p.HoraInicio, &p.HoraFin, &p.Recurrencia); err != nil {
			return nil, err
		}
		programas = append(programas, p)
	}

	return programas, nil
}

// CrearProgramaRonda inserta un nuevo programa de patrullaje y sus puntos asociados
func CrearProgramaRonda(ctx context.Context, p *model.ProgramaRonda) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO programas_rondas (residencial_id, nombre, hora_inicio, hora_fin, recurrencia) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err = tx.QueryRowContext(ctx, query, p.ResidencialID, p.Nombre, p.HoraInicio, p.HoraFin, p.Recurrencia).Scan(&p.ID)
	if err != nil {
		return err
	}

	// Insertar puntos si existen
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

// CrearAsignacionRonda asigna un programa a un usuario en una fecha
func CrearAsignacionRonda(ctx context.Context, a *model.AsignacionRonda) error {
	query := `INSERT INTO asignaciones_rondas (residencial_id, programa_id, usuario_id, fecha, estado) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return db.DB.QueryRowContext(ctx, query, a.ResidencialID, a.ProgramaID, a.UsuarioID, a.Fecha, "POR_ASIGNAR").Scan(&a.ID)
}

// ObtenerAsignacionesRondas devuelve las asignaciones de rondas para una fecha específica o todas si no hay fecha
func ObtenerAsignacionesRondas(ctx context.Context, residencialID string) ([]model.AsignacionRonda, error) {
	rows, err := db.DB.QueryContext(ctx, "SELECT id, residencial_id, programa_id, COALESCE(usuario_id, 0), fecha, estado FROM asignaciones_rondas WHERE residencial_id = $1", residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var asignaciones []model.AsignacionRonda
	for rows.Next() {
		var a model.AsignacionRonda
		if err := rows.Scan(&a.ID, &a.ResidencialID, &a.ProgramaID, &a.UsuarioID, &a.Fecha, &a.Estado); err != nil {
			return nil, err
		}
		asignaciones = append(asignaciones, a)
	}

	return asignaciones, nil
}
