package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"time"
)

// InitIncidentesDB crea la tabla si no existe
func InitIncidentesDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS incidentes (
		id SERIAL PRIMARY KEY,
		residencial_id VARCHAR(50) NOT NULL,
		reportado_por VARCHAR(100) NOT NULL,
		tipo VARCHAR(50) NOT NULL,
		descripcion TEXT NOT NULL,
		nivel_gravedad VARCHAR(20) NOT NULL,
		estado VARCHAR(20) NOT NULL DEFAULT 'ABIERTO',
		evidencia_url TEXT,
		ubicacion VARCHAR(150),
		fecha_creacion TIMESTAMP NOT NULL DEFAULT NOW(),
		fecha_resolucion TIMESTAMP
	);
	ALTER TABLE incidentes ADD COLUMN IF NOT EXISTS ubicacion VARCHAR(150);
	CREATE INDEX IF NOT EXISTS idx_incidentes_residencial ON incidentes(residencial_id);
	`
	_, err := db.DB.Exec(query)
	return err
}

func CrearIncidente(ctx context.Context, incidente *model.Incidente) error {
	query := `
		INSERT INTO incidentes (residencial_id, reportado_por, tipo, descripcion, nivel_gravedad, estado, evidencia_url, ubicacion, fecha_creacion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	incidente.FechaCreacion = time.Now()
	if incidente.Estado == "" {
		incidente.Estado = "ABIERTO"
	}

	err := db.DB.QueryRowContext(ctx, query,
		incidente.ResidencialID,
		incidente.ReportadoPor,
		incidente.Tipo,
		incidente.Descripcion,
		incidente.NivelGravedad,
		incidente.Estado,
		incidente.EvidenciaURL,
		incidente.Ubicacion,
		incidente.FechaCreacion,
	).Scan(&incidente.ID)
	
	if err != nil {
		return err
	}

	// Trigger audit log asynchronously
	go RegistrarAuditoriaAsync(
		incidente.ResidencialID,
		incidente.ReportadoPor, // Guarding the user ID conceptually
		"CREAR_INCIDENTE",
		"Incidente creado",
		"NA",
	)

	return nil
}

func ObtenerIncidentes(ctx context.Context, residencialID string) ([]model.Incidente, error) {
	query := `
		SELECT id, residencial_id, reportado_por, tipo, descripcion, nivel_gravedad, estado, COALESCE(evidencia_url, ''), COALESCE(ubicacion, ''), fecha_creacion, fecha_resolucion
		FROM incidentes
		WHERE residencial_id = $1
		ORDER BY fecha_creacion DESC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidentes []model.Incidente
	for rows.Next() {
		var i model.Incidente
		if err := rows.Scan(
			&i.ID,
			&i.ResidencialID,
			&i.ReportadoPor,
			&i.Tipo,
			&i.Descripcion,
			&i.NivelGravedad,
			&i.Estado,
			&i.EvidenciaURL,
			&i.Ubicacion,
			&i.FechaCreacion,
			&i.FechaResolucion,
		); err != nil {
			return nil, err
		}
		incidentes = append(incidentes, i)
	}

	// Trigger audit log asynchronously
	go RegistrarAuditoriaAsync(
		residencialID,
		"SISTEMA", // Or the requester if available
		"CONSULTA_INCIDENTES",
		"Consulta lista de incidentes",
		"NA",
	)

	return incidentes, nil
}

func ResolverIncidente(ctx context.Context, residencialID string, id int64) error {
	query := `
		UPDATE incidentes
		SET estado = 'RESUELTO', fecha_resolucion = $1
		WHERE id = $2 AND residencial_id = $3
	`
	_, err := db.DB.ExecContext(ctx, query, time.Now(), id, residencialID)
	
	if err == nil {
		go RegistrarAuditoriaAsync(
			residencialID,
			"SISTEMA",
			"RESOLVER_INCIDENTE",
			"Incidente resuelto",
			"NA",
		)
	}
	return err
}
