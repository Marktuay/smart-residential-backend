package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// ObtenerVisitas retorna todas las visitas de un residencial
func ObtenerVisitas(ctx context.Context, residencialID string) ([]model.Visita, error) {
	query := `
		SELECT id, residencial_id, nombre_visitante, cedula_visitante, estado, fecha_esperada, numero_casa, medio_ingreso, placa_vehiculo, fecha_ingreso, fecha_salida
		FROM visitas
		WHERE residencial_id = $1
		ORDER BY id DESC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visitas []model.Visita
	for rows.Next() {
		var v model.Visita
		err := rows.Scan(
			&v.ID, &v.ResidencialID, &v.NombreVisitante, &v.CedulaVisitante, &v.Estado, &v.FechaEsperada, &v.NumeroCasa, &v.MedioIngreso, &v.PlacaVehiculo, &v.FechaIngreso, &v.FechaSalida,
		)
		if err != nil {
			return nil, err
		}
		visitas = append(visitas, v)
	}
	return visitas, nil
}

// RegistrarVisita inserta una nueva visita esperada
func RegistrarVisita(ctx context.Context, v *model.Visita) error {
	query := `
		INSERT INTO visitas (residencial_id, nombre_visitante, cedula_visitante, estado, fecha_esperada, numero_casa, medio_ingreso, placa_vehiculo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	return db.DB.QueryRowContext(ctx, query,
		v.ResidencialID,
		v.NombreVisitante,
		v.CedulaVisitante,
		"PENDIENTE",
		v.FechaEsperada,
		v.NumeroCasa,
		v.MedioIngreso,
		v.PlacaVehiculo,
	).Scan(&v.ID)
}

// ActualizarEstadoVisita actualiza el estado y los tiempos de la visita
func ActualizarEstadoVisita(ctx context.Context, residencialID string, id int64, estado string) error {
	var query string
	if estado == "EN_CURSO" {
		query = `
			UPDATE visitas 
			SET estado = $1, fecha_ingreso = CURRENT_TIMESTAMP 
			WHERE residencial_id = $2 AND id = $3
		`
	} else if estado == "FINALIZADA" {
		query = `
			UPDATE visitas 
			SET estado = $1, fecha_salida = CURRENT_TIMESTAMP 
			WHERE residencial_id = $2 AND id = $3
		`
	} else {
		query = `
			UPDATE visitas 
			SET estado = $1 
			WHERE residencial_id = $2 AND id = $3
		`
	}
	_, err := db.DB.ExecContext(ctx, query, estado, residencialID, id)
	return err
}
