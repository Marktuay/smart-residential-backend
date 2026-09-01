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

// ListarUsuarios obtiene todos los usuarios de un residencial
func ListarUsuarios(residencialID string) ([]model.Usuario, error) {
	query := `SELECT id, residencial_id, email, rol, creado_en 
			  FROM usuarios 
			  WHERE residencial_id = $1 
			  ORDER BY creado_en DESC`

	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usuarios []model.Usuario
	for rows.Next() {
		var u model.Usuario
		if err := rows.Scan(&u.ID, &u.ResidencialID, &u.Email, &u.Rol, &u.CreadoEn); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, u)
	}
	return usuarios, nil
}

// CrearUsuario inserta un nuevo usuario en la base de datos
func CrearUsuario(usuario *model.Usuario) error {
	query := `INSERT INTO usuarios (residencial_id, email, password_hash, rol) 
			  VALUES ($1, $2, $3, $4) RETURNING id, creado_en`
	
	err := db.DB.QueryRow(query,
		usuario.ResidencialID,
		usuario.Email,
		usuario.PasswordHash,
		usuario.Rol,
	).Scan(&usuario.ID, &usuario.CreadoEn)

	return err
}

// ActualizarUsuario actualiza los datos de un usuario (email, rol y opcionalmente password)
func ActualizarUsuario(id int64, email, rol, passwordHash string) error {
	if passwordHash != "" {
		query := `UPDATE usuarios SET email = $1, rol = $2, password_hash = $3 WHERE id = $4`
		_, err := db.DB.Exec(query, email, rol, passwordHash, id)
		return err
	}
	query := `UPDATE usuarios SET email = $1, rol = $2 WHERE id = $3`
	_, err := db.DB.Exec(query, email, rol, id)
	return err
}

// EliminarUsuario elimina un usuario por su ID
func EliminarUsuario(id int64) error {
	query := `DELETE FROM usuarios WHERE id = $1`
	_, err := db.DB.Exec(query, id)
	return err
}
