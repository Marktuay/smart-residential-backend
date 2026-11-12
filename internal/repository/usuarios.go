package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitUsuariosDB garantiza que la tabla usuarios y sus nuevas columnas existan
func InitUsuariosDB() error {
	queries := []string{
		`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS nombre_completo VARCHAR(255) DEFAULT '';`,
		`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS telefono VARCHAR(50) DEFAULT '';`,
		`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS cedula VARCHAR(50) DEFAULT '';`,
		`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT TRUE;`,
		`ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_usuarios_residencial_del ON usuarios(residencial_id, deleted_at);`,
	}
	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error al actualizar esquema de usuarios: %v", err)
		}
	}
	return nil
}

// ObtenerUsuarioPorEmail busca un usuario activo por su email/username para el login
func ObtenerUsuarioPorEmail(ctx context.Context, email string) (*model.Usuario, error) {
	cleanEmail := strings.TrimSpace(email)
	query := `SELECT id, residencial_id, email, COALESCE(nombre_completo, ''), COALESCE(telefono, ''), COALESCE(cedula, ''), password_hash, rol, COALESCE(activo, true), creado_en 
			  FROM usuarios 
			  WHERE LOWER(TRIM(email)) = LOWER($1) AND deleted_at IS NULL AND COALESCE(activo, true) = true`

	var u model.Usuario
	err := db.DB.QueryRowContext(ctx, query, cleanEmail).Scan(
		&u.ID, &u.ResidencialID, &u.Email, &u.NombreCompleto, &u.Telefono, &u.Cedula, &u.PasswordHash, &u.Rol, &u.Activo, &u.CreadoEn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Usuario no encontrado
		}
		return nil, err
	}

	return &u, nil
}

// ListarUsuarios obtiene todos los usuarios activos de un residencial (compatibilidad hacia atrás)
func ListarUsuarios(residencialID string) ([]model.Usuario, error) {
	usuarios, _, err := ListarUsuariosFiltro(residencialID, "", "", nil, 0, 0)
	return usuarios, err
}

// ListarUsuariosFiltro obtiene usuarios con búsqueda, filtrado por rol/estado y paginación
func ListarUsuariosFiltro(residencialID string, search string, rol string, activo *bool, page int, limit int) ([]model.Usuario, int, error) {
	baseWhere := "WHERE residencial_id = $1 AND deleted_at IS NULL"
	args := []interface{}{residencialID}
	argIdx := 2

	if strings.TrimSpace(search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		baseWhere += fmt.Sprintf(" AND (LOWER(email) LIKE $%d OR LOWER(COALESCE(nombre_completo, '')) LIKE $%d OR LOWER(COALESCE(cedula, '')) LIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, s)
		argIdx++
	}

	if strings.TrimSpace(rol) != "" && rol != "TODOS" {
		baseWhere += fmt.Sprintf(" AND rol = $%d", argIdx)
		args = append(args, strings.TrimSpace(rol))
		argIdx++
	}

	if activo != nil {
		baseWhere += fmt.Sprintf(" AND activo = $%d", argIdx)
		args = append(args, *activo)
		argIdx++
	}

	// Contar total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM usuarios %s", baseWhere)
	var total int
	err := db.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Consulta de datos
	dataQuery := fmt.Sprintf(`SELECT id, residencial_id, email, COALESCE(nombre_completo, ''), COALESCE(telefono, ''), COALESCE(cedula, ''), rol, COALESCE(activo, true), creado_en, deleted_at 
			  FROM usuarios 
			  %s 
			  ORDER BY creado_en DESC`, baseWhere)

	if limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++

		offset := 0
		if page > 1 {
			offset = (page - 1) * limit
		}
		dataQuery += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
		argIdx++
	}

	rows, err := db.DB.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var usuarios []model.Usuario
	for rows.Next() {
		var u model.Usuario
		if err := rows.Scan(
			&u.ID, &u.ResidencialID, &u.Email, &u.NombreCompleto, &u.Telefono, &u.Cedula, &u.Rol, &u.Activo, &u.CreadoEn, &u.DeletedAt,
		); err != nil {
			return nil, 0, err
		}
		usuarios = append(usuarios, u)
	}

	if usuarios == nil {
		usuarios = []model.Usuario{}
	}

	return usuarios, total, nil
}

// CrearUsuario inserta un nuevo usuario en la base de datos
func CrearUsuario(usuario *model.Usuario) error {
	query := `INSERT INTO usuarios (residencial_id, email, nombre_completo, telefono, cedula, password_hash, rol, activo) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, creado_en`
	
	return db.DB.QueryRow(query,
		usuario.ResidencialID,
		usuario.Email,
		usuario.NombreCompleto,
		usuario.Telefono,
		usuario.Cedula,
		usuario.PasswordHash,
		usuario.Rol,
		usuario.Activo,
	).Scan(&usuario.ID, &usuario.CreadoEn)
}

// ActualizarUsuario actualiza los datos de un usuario (email, nombre, telefono, cedula, rol, activo y opcionalmente password)
func ActualizarUsuario(id int64, residencialID string, email, nombreCompleto, telefono, cedula, rol string, activo bool, passwordHash string) error {
	if passwordHash != "" {
		query := `UPDATE usuarios 
				  SET email = $1, nombre_completo = $2, telefono = $3, cedula = $4, rol = $5, activo = $6, password_hash = $7 
				  WHERE id = $8 AND residencial_id = $9 AND deleted_at IS NULL`
		_, err := db.DB.Exec(query, email, nombreCompleto, telefono, cedula, rol, activo, passwordHash, id, residencialID)
		return err
	}
	query := `UPDATE usuarios 
			  SET email = $1, nombre_completo = $2, telefono = $3, cedula = $4, rol = $5, activo = $6 
			  WHERE id = $7 AND residencial_id = $8 AND deleted_at IS NULL`
	_, err := db.DB.Exec(query, email, nombreCompleto, telefono, cedula, rol, activo, id, residencialID)
	return err
}

// CambiarEstadoUsuario activa o desactiva un usuario
func CambiarEstadoUsuario(id int64, residencialID string, activo bool) error {
	query := `UPDATE usuarios SET activo = $1 WHERE id = $2 AND residencial_id = $3 AND deleted_at IS NULL`
	_, err := db.DB.Exec(query, activo, id, residencialID)
	return err
}

// EliminarUsuario realiza un SoftDelete marcando deleted_at y activo = false
func EliminarUsuario(id int64, residencialID string) error {
	query := `UPDATE usuarios SET deleted_at = NOW(), activo = false WHERE id = $1 AND residencial_id = $2 AND deleted_at IS NULL`
	res, err := db.DB.Exec(query, id, residencialID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("usuario no encontrado o ya eliminado")
	}
	return nil
}

// CrearUsuariosBatch procesa la carga masiva en bloque dentro de una transacción
func CrearUsuariosBatch(residencialID string, usuarios []model.Usuario) (int, error) {
	if len(usuarios) == 0 {
		return 0, nil
	}
	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO usuarios (residencial_id, email, nombre_completo, telefono, cedula, password_hash, rol, activo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET
			nombre_completo = EXCLUDED.nombre_completo,
			telefono = EXCLUDED.telefono,
			cedula = EXCLUDED.cedula,
			rol = EXCLUDED.rol,
			activo = EXCLUDED.activo,
			deleted_at = NULL
		WHERE usuarios.residencial_id = EXCLUDED.residencial_id
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	insertados := 0
	for _, u := range usuarios {
		_, err := stmt.Exec(residencialID, u.Email, u.NombreCompleto, u.Telefono, u.Cedula, u.PasswordHash, u.Rol, u.Activo)
		if err != nil {
			return insertados, err
		}
		insertados++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return insertados, nil
}
