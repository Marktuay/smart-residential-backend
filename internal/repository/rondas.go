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

// CrearProgramaRonda inserta un nuevo programa de patrullaje
func CrearProgramaRonda(ctx context.Context, p *model.ProgramaRonda) error {
	query := `INSERT INTO programas_rondas (residencial_id, nombre, hora_inicio, hora_fin, recurrencia) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return db.DB.QueryRowContext(ctx, query, p.ResidencialID, p.Nombre, p.HoraInicio, p.HoraFin, p.Recurrencia).Scan(&p.ID)
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
