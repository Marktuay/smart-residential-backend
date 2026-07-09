package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// CrearIncidente inserta un nuevo incidente en la base de datos
func CrearIncidente(ctx context.Context, inc *model.Incidente) error {
	query := `INSERT INTO incidentes (residencial_id, titulo, descripcion, prioridad, estado, categoria_id, creado_en) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	
	err := db.DB.QueryRowContext(ctx, query,
		inc.ResidencialID, inc.Titulo, inc.Descripcion, inc.Prioridad, inc.Estado, inc.CategoriaID, inc.CreadoEn).Scan(&inc.ID)
	
	return err
}

// ListarIncidentes obtiene los incidentes de un residencial
func ListarIncidentes(ctx context.Context, residencialID string) ([]model.Incidente, error) {
	query := `SELECT id, titulo, descripcion, prioridad, estado, categoria_id, creado_en 
			  FROM incidentes WHERE residencial_id = $1 ORDER BY creado_en DESC`
	
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidentes []model.Incidente
	for rows.Next() {
		var inc model.Incidente
		if err := rows.Scan(&inc.ID, &inc.Titulo, &inc.Descripcion, &inc.Prioridad, &inc.Estado, &inc.CategoriaID, &inc.CreadoEn); err != nil {
			return nil, err
		}
		inc.ResidencialID = residencialID
		incidentes = append(incidentes, inc)
	}
	
	if incidentes == nil {
		incidentes = []model.Incidente{}
	}
	
	return incidentes, nil
}
