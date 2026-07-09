package repository

import (
	"context"
	"database/sql"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// ObtenerUsuarioPorEmail busca un usuario por su email para el login
func ObtenerUsuarioPorEmail(ctx context.Context, email string) (*model.Usuario, error) {
	query := `SELECT id, residencial_id, email, password_hash, rol, creado_en 
			  FROM usuarios WHERE email = $1`

	var u model.Usuario
	err := db.DB.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.ResidencialID, &u.Email, &u.PasswordHash, &u.Rol, &u.CreadoEn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Usuario no encontrado
		}
		return nil, err
	}

	return &u, nil
}
