package repository

import (
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

func ObtenerCasas(residencialID string) ([]model.Casa, error) {
	query := `
		SELECT id, residencial_id, numero_casa, bloque, estado, created_at
		FROM casas
		WHERE residencial_id = $1
		ORDER BY bloque, numero_casa ASC`

	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var casas []model.Casa
	for rows.Next() {
		var c model.Casa
		if err := rows.Scan(&c.ID, &c.ResidencialID, &c.NumeroCasa, &c.Bloque, &c.Estado, &c.CreatedAt); err != nil {
			return nil, err
		}
		casas = append(casas, c)
	}
	// If casas is nil, return an empty array instead of null
	if casas == nil {
		casas = []model.Casa{}
	}
	return casas, nil
}

func RegistrarCasa(casa model.Casa) (int64, error) {
	query := `
		INSERT INTO casas (residencial_id, numero_casa, bloque, estado)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := db.DB.QueryRow(query, casa.ResidencialID, casa.NumeroCasa, casa.Bloque, casa.Estado).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func ObtenerResidentes(residencialID string) ([]model.Residente, error) {
	query := `
		SELECT r.id, r.residencial_id, r.casa_id, r.nombre, r.telefono, r.email, r.es_propietario, r.created_at, c.numero_casa
		FROM residentes r
		LEFT JOIN casas c ON r.casa_id = c.id
		WHERE r.residencial_id = $1
		ORDER BY r.nombre ASC`

	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var residentes []model.Residente
	for rows.Next() {
		var r model.Residente
		var numeroCasa *string
		if err := rows.Scan(&r.ID, &r.ResidencialID, &r.CasaID, &r.Nombre, &r.Telefono, &r.Email, &r.EsPropietario, &r.CreatedAt, &numeroCasa); err != nil {
			return nil, err
		}
		if numeroCasa != nil {
			r.NumeroCasa = *numeroCasa
		}
		residentes = append(residentes, r)
	}
	// If residentes is nil, return an empty array instead of null
	if residentes == nil {
		residentes = []model.Residente{}
	}
	return residentes, nil
}

func RegistrarResidente(residente model.Residente) (int64, error) {
	query := `
		INSERT INTO residentes (residencial_id, casa_id, nombre, telefono, email, es_propietario)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := db.DB.QueryRow(query, residente.ResidencialID, residente.CasaID, residente.Nombre, residente.Telefono, residente.Email, residente.EsPropietario).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
