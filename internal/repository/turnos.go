package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitTurnosDB crea la tabla si no existe
func InitTurnosDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS turnos (
		id SERIAL PRIMARY KEY,
		residencial_id VARCHAR(50) NOT NULL,
		guardia_id INT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
		fecha DATE NOT NULL,
		hora_inicio TIME NOT NULL,
		hora_fin TIME NOT NULL,
		descripcion TEXT,
		ubicacion VARCHAR(150),
		estado VARCHAR(50) NOT NULL DEFAULT 'PROGRAMADO'
	);
	CREATE INDEX IF NOT EXISTS idx_turnos_residencial ON turnos(residencial_id);
	`
	_, err := db.DB.Exec(query)
	return err
}

// ObtenerTurnos retorna todos los turnos activos de un residencial
func ObtenerTurnos(ctx context.Context, residencialID string) ([]model.Turno, error) {
	query := `
		SELECT t.id, t.residencial_id, t.guardia_id, u.email, 
		       TO_CHAR(t.fecha, 'YYYY-MM-DD'), 
		       TO_CHAR(t.hora_inicio, 'HH24:MI'), 
		       TO_CHAR(t.hora_fin, 'HH24:MI'), 
		       COALESCE(t.descripcion, ''), 
		       COALESCE(t.ubicacion, ''), 
		       t.estado
		FROM turnos t
		LEFT JOIN usuarios u ON t.guardia_id = u.id
		WHERE t.residencial_id = $1
		ORDER BY t.fecha DESC, t.hora_inicio ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turnos []model.Turno
	for rows.Next() {
		var t model.Turno
		err := rows.Scan(
			&t.ID, &t.ResidencialID, &t.GuardiaID, &t.GuardiaEmail,
			&t.Fecha, &t.HoraInicio, &t.HoraFin, &t.Descripcion,
			&t.Ubicacion, &t.Estado,
		)
		if err != nil {
			return nil, err
		}
		turnos = append(turnos, t)
	}

	if turnos == nil {
		turnos = []model.Turno{}
	}

	return turnos, nil
}

// CrearTurno inserta un nuevo turno
func CrearTurno(ctx context.Context, t *model.Turno) error {
	query := `
		INSERT INTO turnos (residencial_id, guardia_id, fecha, hora_inicio, hora_fin, descripcion, ubicacion, estado)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	if t.Estado == "" {
		t.Estado = "PROGRAMADO"
	}

	err := db.DB.QueryRowContext(ctx, query,
		t.ResidencialID,
		t.GuardiaID,
		t.Fecha,
		t.HoraInicio,
		t.HoraFin,
		t.Descripcion,
		t.Ubicacion,
		t.Estado,
	).Scan(&t.ID)

	if err == nil {
		go RegistrarAuditoriaAsync(
			t.ResidencialID,
			"CREAR_TURNO",
			"SISTEMA", // Se puede actualizar en el handler con el actor real
			"NA",      // Se puede actualizar con la IP
			"Turno creado para guardia ID: "+strconvFormat(t.GuardiaID)+" en fecha: "+t.Fecha,
		)
	}

	return err
}

// ActualizarTurno actualiza un turno existente
func ActualizarTurno(ctx context.Context, t *model.Turno) error {
	query := `
		UPDATE turnos
		SET guardia_id = $1, fecha = $2, hora_inicio = $3, hora_fin = $4, descripcion = $5, ubicacion = $6, estado = $7
		WHERE id = $8 AND residencial_id = $9
	`
	_, err := db.DB.ExecContext(ctx, query,
		t.GuardiaID,
		t.Fecha,
		t.HoraInicio,
		t.HoraFin,
		t.Descripcion,
		t.Ubicacion,
		t.Estado,
		t.ID,
		t.ResidencialID,
	)

	if err == nil {
		go RegistrarAuditoriaAsync(
			t.ResidencialID,
			"ACTUALIZAR_TURNO",
			"SISTEMA",
			"NA",
			"Turno ID "+strconvFormat(t.ID)+" actualizado. Guardia: "+strconvFormat(t.GuardiaID),
		)
	}

	return err
}

// EliminarTurno elimina un turno (soft-delete cambiándolo a CANCELADO)
func EliminarTurno(ctx context.Context, id int64, residencialID string) error {
	query := `
		UPDATE turnos
		SET estado = 'CANCELADO'
		WHERE id = $1 AND residencial_id = $2
	`
	_, err := db.DB.ExecContext(ctx, query, id, residencialID)

	if err == nil {
		go RegistrarAuditoriaAsync(
			residencialID,
			"ELIMINAR_TURNO",
			"SISTEMA",
			"NA",
			"Turno ID "+strconvFormat(id)+" cancelado",
		)
	}

	return err
}

// helper strconvFormat
func strconvFormat(i int64) string {
	// Simple convert
	var s string
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
