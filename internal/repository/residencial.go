package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitResidencialesDB crea la tabla si no existe
func InitResidencialesDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS residenciales (
		id VARCHAR(50) PRIMARY KEY,
		nombre VARCHAR(100) NOT NULL,
		direccion TEXT,
		latitud_centro DOUBLE PRECISION DEFAULT 12.1364,
		longitud_centro DOUBLE PRECISION DEFAULT -86.2514,
		zoom_defecto INTEGER DEFAULT 13,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`
	_, err := db.DB.Exec(query)
	if err != nil {
		return err
	}

	alterQueries := []string{
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS tolerancia_ronda_mins INTEGER DEFAULT 15`,
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS frecuencia_gps_seg INTEGER DEFAULT 10`,
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS email_alerta_incidentes VARCHAR(150) DEFAULT ''`,
		`ALTER TABLE residenciales ADD COLUMN IF NOT EXISTS telefono_emergencia VARCHAR(20) DEFAULT ''`,
	}
	for _, q := range alterQueries {
		if _, err := db.DB.Exec(q); err != nil {
			return err
		}
	}

	return InitCasasResidentesDB()
}

// InitCasasResidentesDB inicializa tablas de casas, residentes y asignaciones N:M
func InitCasasResidentesDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS casas (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			numero_casa VARCHAR(50) NOT NULL,
			bloque VARCHAR(50) DEFAULT '',
			estado VARCHAR(30) DEFAULT 'OCUPADA',
			tiene_contrato_seguridad BOOLEAN DEFAULT FALSE,
			activo BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP NULL
		);`,
		`ALTER TABLE casas ADD COLUMN IF NOT EXISTS tiene_contrato_seguridad BOOLEAN DEFAULT FALSE;`,
		`ALTER TABLE casas ADD COLUMN IF NOT EXISTS activo BOOLEAN DEFAULT TRUE;`,
		`ALTER TABLE casas ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_casas_residencial ON casas(residencial_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_casas_unique_activa 
		 ON casas (residencial_id, LOWER(TRIM(numero_casa)), LOWER(TRIM(COALESCE(bloque, '')))) 
		 WHERE deleted_at IS NULL;`,

		`CREATE TABLE IF NOT EXISTS residentes (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			casa_id BIGINT NULL,
			nombre VARCHAR(150) NOT NULL,
			cedula VARCHAR(50) DEFAULT '',
			telefono VARCHAR(50) DEFAULT '',
			telefono_emergencia VARCHAR(50) DEFAULT '',
			email VARCHAR(150) DEFAULT '',
			es_propietario BOOLEAN DEFAULT TRUE,
			activo BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP NULL
		);`,
		`ALTER TABLE residentes ADD COLUMN IF NOT EXISTS cedula VARCHAR(50) DEFAULT '';`,
		`ALTER TABLE residentes ADD COLUMN IF NOT EXISTS telefono_emergencia VARCHAR(50) DEFAULT '';`,
		`ALTER TABLE residentes ADD COLUMN IF NOT EXISTS activo BOOLEAN DEFAULT TRUE;`,
		`ALTER TABLE residentes ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_residentes_residencial ON residentes(residencial_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_residentes_unique_cedula
		 ON residentes (residencial_id, LOWER(TRIM(cedula)))
		 WHERE deleted_at IS NULL AND cedula <> '';`,

		`CREATE TABLE IF NOT EXISTS residente_casas (
			id BIGSERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			residente_id BIGINT NOT NULL,
			casa_id BIGINT NOT NULL,
			es_principal BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_residente_casa UNIQUE(residencial_id, residente_id, casa_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_rc_residente ON residente_casas(residente_id);`,
		`CREATE INDEX IF NOT EXISTS idx_rc_casa ON residente_casas(casa_id);`,
	}

	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error al ejecutar migración casas/residentes: %w", err)
		}
	}
	return nil
}

func ListarResidenciales(ctx context.Context) ([]model.Residencial, error) {
	query := `
		SELECT id, nombre, direccion, latitud_centro, longitud_centro, zoom_defecto, 
		       tolerancia_ronda_mins, frecuencia_gps_seg, email_alerta_incidentes, telefono_emergencia, created_at
		FROM residenciales
		ORDER BY nombre ASC`

	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var residenciales []model.Residencial
	for rows.Next() {
		var r model.Residencial
		if err := rows.Scan(
			&r.ID, &r.Nombre, &r.Direccion,
			&r.LatitudCentro, &r.LongitudCentro, &r.ZoomDefecto,
			&r.ToleranciaRondaMins, &r.FrecuenciaGpsSeg, &r.EmailAlertaIncidentes, &r.TelefonoEmergencia,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		residenciales = append(residenciales, r)
	}

	if residenciales == nil {
		residenciales = []model.Residencial{}
	}
	return residenciales, nil
}

func CrearResidencial(ctx context.Context, r *model.Residencial) error {
	query := `
		INSERT INTO residenciales (id, nombre, direccion, latitud_centro, longitud_centro, zoom_defecto)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.DB.ExecContext(ctx, query,
		r.ID, r.Nombre, r.Direccion,
		r.LatitudCentro, r.LongitudCentro, r.ZoomDefecto,
	)
	return err
}

func ObtenerResidencial(id string) (*model.Residencial, error) {
	query := `
		SELECT id, nombre, direccion, latitud_centro, longitud_centro, zoom_defecto, 
		       tolerancia_ronda_mins, frecuencia_gps_seg, email_alerta_incidentes, telefono_emergencia, created_at
		FROM residenciales
		WHERE id = $1`

	var r model.Residencial
	err := db.DB.QueryRow(query, id).Scan(
		&r.ID, &r.Nombre, &r.Direccion, 
		&r.LatitudCentro, &r.LongitudCentro, &r.ZoomDefecto, 
		&r.ToleranciaRondaMins, &r.FrecuenciaGpsSeg, &r.EmailAlertaIncidentes, &r.TelefonoEmergencia,
		&r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ActualizarConfiguracionResidencial(ctx context.Context, id string, lat float64, lng float64, zoom int, tolerancia int, frecuencia int, email string, tel string) error {
	query := `
		UPDATE residenciales
		SET latitud_centro = $1,
		    longitud_centro = $2,
		    zoom_defecto = $3,
		    tolerancia_ronda_mins = $4,
		    frecuencia_gps_seg = $5,
		    email_alerta_incidentes = $6,
		    telefono_emergencia = $7
		WHERE id = $8
	`
	_, err := db.DB.ExecContext(ctx, query, lat, lng, zoom, tolerancia, frecuencia, email, tel, id)
	return err
}

// ==========================================
// SECCIÓN DE CASAS / VIVIENDAS
// ==========================================

// ObtenerCasas retorna todas las casas activas (para selectores y compatibilidad)
func ObtenerCasas(residencialID string) ([]model.Casa, error) {
	query := `
		SELECT c.id, c.residencial_id, c.numero_casa, c.bloque, c.estado, 
		       COALESCE(c.tiene_contrato_seguridad, false),
		       COALESCE(string_agg(res.nombre, ', '), '') AS residentes_nombres,
		       COUNT(rc.residente_id) AS residentes_count,
		       COALESCE(c.activo, true),
		       TO_CHAR(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM casas c
		LEFT JOIN residente_casas rc ON c.id = rc.casa_id AND rc.residencial_id = c.residencial_id
		LEFT JOIN residentes res ON rc.residente_id = res.id AND res.deleted_at IS NULL
		WHERE c.residencial_id = $1 AND c.deleted_at IS NULL
		GROUP BY c.id, c.residencial_id, c.numero_casa, c.bloque, c.estado, c.tiene_contrato_seguridad, c.activo, c.created_at
		ORDER BY c.bloque ASC, c.numero_casa ASC`

	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var casas []model.Casa
	for rows.Next() {
		var c model.Casa
		if err := rows.Scan(
			&c.ID, &c.ResidencialID, &c.NumeroCasa, &c.Bloque, &c.Estado,
			&c.TieneContratoSeguridad, &c.ResidentesNombres, &c.ResidentesCount,
			&c.Activo, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		casas = append(casas, c)
	}
	if casas == nil {
		casas = []model.Casa{}
	}
	return casas, nil
}

// ListarCasasPaginadas filtra y pagina casas con búsqueda en servidor
func ListarCasasPaginadas(residencialID string, search string, bloque string, estado string, page int, limit int) (*model.CasaPaginadaResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	whereClauses := []string{"c.residencial_id = $1", "c.deleted_at IS NULL"}
	args := []interface{}{residencialID}
	argIdx := 2

	if strings.TrimSpace(search) != "" {
		s := "%" + strings.TrimSpace(search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(c.numero_casa ILIKE $%d OR c.bloque ILIKE $%d OR res.nombre ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, s)
		argIdx++
	}

	if strings.TrimSpace(bloque) != "" && bloque != "TODOS" {
		whereClauses = append(whereClauses, fmt.Sprintf("c.bloque = $%d", argIdx))
		args = append(args, strings.TrimSpace(bloque))
		argIdx++
	}

	if strings.TrimSpace(estado) != "" && estado != "TODOS" {
		whereClauses = append(whereClauses, fmt.Sprintf("c.estado = $%d", argIdx))
		args = append(args, strings.TrimSpace(estado))
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Conteo total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT c.id)
		FROM casas c
		LEFT JOIN residente_casas rc ON c.id = rc.casa_id AND rc.residencial_id = c.residencial_id
		LEFT JOIN residentes res ON rc.residente_id = res.id AND res.deleted_at IS NULL
		WHERE %s`, whereSQL)

	var total int
	if err := db.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("error al contar casas: %w", err)
	}

	// Consulta de datos
	dataQuery := fmt.Sprintf(`
		SELECT c.id, c.residencial_id, c.numero_casa, c.bloque, c.estado, 
		       COALESCE(c.tiene_contrato_seguridad, false),
		       COALESCE(string_agg(res.nombre, ', '), '') AS residentes_nombres,
		       COUNT(DISTINCT rc.residente_id) AS residentes_count,
		       COALESCE(c.activo, true),
		       TO_CHAR(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM casas c
		LEFT JOIN residente_casas rc ON c.id = rc.casa_id AND rc.residencial_id = c.residencial_id
		LEFT JOIN residentes res ON rc.residente_id = res.id AND res.deleted_at IS NULL
		WHERE %s
		GROUP BY c.id, c.residencial_id, c.numero_casa, c.bloque, c.estado, c.tiene_contrato_seguridad, c.activo, c.created_at
		ORDER BY c.bloque ASC, c.numero_casa ASC
		LIMIT $%d OFFSET $%d`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, limit, offset)
	rows, err := db.DB.Query(dataQuery, argsWithPaging...)
	if err != nil {
		return nil, fmt.Errorf("error al listar casas paginadas: %w", err)
	}
	defer rows.Close()

	var casas []model.Casa
	for rows.Next() {
		var c model.Casa
		if err := rows.Scan(
			&c.ID, &c.ResidencialID, &c.NumeroCasa, &c.Bloque, &c.Estado,
			&c.TieneContratoSeguridad, &c.ResidentesNombres, &c.ResidentesCount,
			&c.Activo, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		casas = append(casas, c)
	}
	if casas == nil {
		casas = []model.Casa{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &model.CasaPaginadaResponse{
		Items:      casas,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// VerificarCasaDuplicada comprueba si ya existe una casa con el mismo número y bloque
func VerificarCasaDuplicada(residencialID string, numeroCasa string, bloque string, excludeID int64) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM casas 
		WHERE residencial_id = $1 
		  AND LOWER(TRIM(numero_casa)) = LOWER(TRIM($2)) 
		  AND LOWER(TRIM(COALESCE(bloque, ''))) = LOWER(TRIM($3))
		  AND deleted_at IS NULL`
	args := []interface{}{residencialID, numeroCasa, bloque}

	if excludeID > 0 {
		query += " AND id <> $4"
		args = append(args, excludeID)
	}

	var count int
	if err := db.DB.QueryRow(query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// RegistrarCasa valida anti-duplicados y crea la casa
func RegistrarCasa(casa model.Casa) (int64, error) {
	if strings.TrimSpace(casa.NumeroCasa) == "" {
		return 0, fmt.Errorf("el número de casa es obligatorio")
	}

	duplicada, err := VerificarCasaDuplicada(casa.ResidencialID, casa.NumeroCasa, casa.Bloque, 0)
	if err != nil {
		return 0, fmt.Errorf("error al verificar duplicidad de casa: %w", err)
	}
	if duplicada {
		return 0, fmt.Errorf("DUPLICATE_CASA: Ya existe la Casa '%s' en el sector/bloque '%s' en este residencial", casa.NumeroCasa, casa.Bloque)
	}

	if casa.Estado == "" {
		casa.Estado = "OCUPADA"
	}

	query := `
		INSERT INTO casas (residencial_id, numero_casa, bloque, estado, tiene_contrato_seguridad, activo)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id`

	var id int64
	err = db.DB.QueryRow(query,
		casa.ResidencialID,
		strings.TrimSpace(casa.NumeroCasa),
		strings.TrimSpace(casa.Bloque),
		casa.Estado,
		casa.TieneContratoSeguridad,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ActualizarCasa modifica los datos de una casa existente validando anti-duplicados
func ActualizarCasa(casa model.Casa) error {
	if casa.ID <= 0 {
		return fmt.Errorf("ID de casa inválido")
	}

	duplicada, err := VerificarCasaDuplicada(casa.ResidencialID, casa.NumeroCasa, casa.Bloque, casa.ID)
	if err != nil {
		return fmt.Errorf("error al verificar duplicidad: %w", err)
	}
	if duplicada {
		return fmt.Errorf("DUPLICATE_CASA: Ya existe otra casa con el número '%s' en el bloque '%s'", casa.NumeroCasa, casa.Bloque)
	}

	query := `
		UPDATE casas
		SET numero_casa = $1, bloque = $2, estado = $3, tiene_contrato_seguridad = $4
		WHERE id = $5 AND residencial_id = $6 AND deleted_at IS NULL`

	res, err := db.DB.Exec(query,
		strings.TrimSpace(casa.NumeroCasa),
		strings.TrimSpace(casa.Bloque),
		casa.Estado,
		casa.TieneContratoSeguridad,
		casa.ID,
		casa.ResidencialID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("casa no encontrada o eliminada")
	}
	return nil
}

// EliminarCasa realiza SoftDelete
func EliminarCasa(id int64, residencialID string) error {
	query := `
		UPDATE casas
		SET deleted_at = NOW(), activo = false
		WHERE id = $1 AND residencial_id = $2 AND deleted_at IS NULL`

	res, err := db.DB.Exec(query, id, residencialID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("casa no encontrada o ya eliminada")
	}
	return nil
}

// ==========================================
// SECCIÓN DE RESIDENTES
// ==========================================

// ObtenerResidentes lista residentes activos (para compatibilidad)
func ObtenerResidentes(residencialID string) ([]model.Residente, error) {
	query := `
		SELECT r.id, r.residencial_id, COALESCE(r.casa_id, 0), r.nombre, 
		       COALESCE(r.cedula, ''), COALESCE(r.telefono, ''), 
		       COALESCE(r.telefono_emergencia, ''), COALESCE(r.email, ''), 
		       r.es_propietario, COALESCE(r.activo, true), r.created_at,
		       COALESCE(c.numero_casa, '')
		FROM residentes r
		LEFT JOIN casas c ON r.casa_id = c.id
		WHERE r.residencial_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.nombre ASC`

	rows, err := db.DB.Query(query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var residentes []model.Residente
	for rows.Next() {
		var r model.Residente
		var dummyCasaID int64
		if err := rows.Scan(
			&r.ID, &r.ResidencialID, &dummyCasaID, &r.Nombre,
			&r.Cedula, &r.Telefono, &r.TelefonoEmergencia, &r.Email,
			&r.EsPropietario, &r.Activo, &r.CreatedAt, &r.NumeroCasa,
		); err != nil {
			return nil, err
		}
		if dummyCasaID > 0 {
			r.CasaID = &dummyCasaID
		}
		residentes = append(residentes, r)
	}
	if residentes == nil {
		residentes = []model.Residente{}
	}

	// Cargar casas asignadas
	for i := range residentes {
		casas, err := ObtenerCasasDeResidente(residencialID, residentes[i].ID)
		if err == nil {
			residentes[i].CasasDetalle = casas
			for _, cs := range casas {
				residentes[i].CasasAsignadasIDs = append(residentes[i].CasasAsignadasIDs, cs.ID)
			}
			if residentes[i].NumeroCasa == "" && len(casas) > 0 {
				residentes[i].NumeroCasa = casas[0].NumeroCasa
			}
		}
	}

	return residentes, nil
}

// ListarResidentesPaginados pagina y busca residentes con filtros
func ListarResidentesPaginados(residencialID string, search string, esPropietario *bool, page int, limit int) (*model.ResidentePaginadoResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	whereClauses := []string{"r.residencial_id = $1", "r.deleted_at IS NULL"}
	args := []interface{}{residencialID}
	argIdx := 2

	if strings.TrimSpace(search) != "" {
		s := "%" + strings.TrimSpace(search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(r.nombre ILIKE $%d OR r.cedula ILIKE $%d OR r.email ILIKE $%d OR r.telefono ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, s)
		argIdx++
	}

	if esPropietario != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("r.es_propietario = $%d", argIdx))
		args = append(args, *esPropietario)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM residentes r WHERE %s`, whereSQL)
	var total int
	if err := db.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("error al contar residentes: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT r.id, r.residencial_id, COALESCE(r.casa_id, 0), r.nombre, 
		       COALESCE(r.cedula, ''), COALESCE(r.telefono, ''), 
		       COALESCE(r.telefono_emergencia, ''), COALESCE(r.email, ''), 
		       r.es_propietario, COALESCE(r.activo, true), r.created_at
		FROM residentes r
		WHERE %s
		ORDER BY r.nombre ASC
		LIMIT $%d OFFSET $%d`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, limit, offset)
	rows, err := db.DB.Query(dataQuery, argsWithPaging...)
	if err != nil {
		return nil, fmt.Errorf("error al listar residentes: %w", err)
	}
	defer rows.Close()

	var residentes []model.Residente
	for rows.Next() {
		var r model.Residente
		var dummyCasaID int64
		if err := rows.Scan(
			&r.ID, &r.ResidencialID, &dummyCasaID, &r.Nombre,
			&r.Cedula, &r.Telefono, &r.TelefonoEmergencia, &r.Email,
			&r.EsPropietario, &r.Activo, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		if dummyCasaID > 0 {
			r.CasaID = &dummyCasaID
		}
		residentes = append(residentes, r)
	}
	if residentes == nil {
		residentes = []model.Residente{}
	}

	// Cargar casas asignadas para cada residente
	for i := range residentes {
		casas, err := ObtenerCasasDeResidente(residencialID, residentes[i].ID)
		if err == nil {
			residentes[i].CasasDetalle = casas
			for _, cs := range casas {
				residentes[i].CasasAsignadasIDs = append(residentes[i].CasasAsignadasIDs, cs.ID)
			}
			if len(casas) > 0 {
				residentes[i].NumeroCasa = casas[0].NumeroCasa
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return &model.ResidentePaginadoResponse{
		Items:      residentes,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// ObtenerCasasDeResidente retorna la lista de casas asociadas a un residente
func ObtenerCasasDeResidente(residencialID string, residenteID int64) ([]model.Casa, error) {
	query := `
		SELECT c.id, c.residencial_id, c.numero_casa, c.bloque, c.estado, 
		       COALESCE(c.tiene_contrato_seguridad, false),
		       COALESCE(c.activo, true),
		       TO_CHAR(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM casas c
		JOIN residente_casas rc ON c.id = rc.casa_id
		WHERE rc.residencial_id = $1 AND rc.residente_id = $2 AND c.deleted_at IS NULL
		ORDER BY rc.es_principal DESC, c.numero_casa ASC`

	rows, err := db.DB.Query(query, residencialID, residenteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var casas []model.Casa
	for rows.Next() {
		var c model.Casa
		if err := rows.Scan(
			&c.ID, &c.ResidencialID, &c.NumeroCasa, &c.Bloque, &c.Estado,
			&c.TieneContratoSeguridad, &c.Activo, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		casas = append(casas, c)
	}
	if casas == nil {
		casas = []model.Casa{}
	}
	return casas, nil
}

// VerificarResidenteDuplicado verifica cédula y correo
func VerificarResidenteDuplicado(residencialID string, cedula string, email string, excludeID int64) (bool, string, error) {
	cedula = strings.TrimSpace(cedula)
	email = strings.TrimSpace(email)

	if cedula != "" {
		query := `SELECT COUNT(*) FROM residentes WHERE residencial_id = $1 AND LOWER(TRIM(cedula)) = LOWER(TRIM($2)) AND deleted_at IS NULL`
		args := []interface{}{residencialID, cedula}
		if excludeID > 0 {
			query += " AND id <> $3"
			args = append(args, excludeID)
		}
		var count int
		if err := db.DB.QueryRow(query, args...).Scan(&count); err != nil {
			return false, "", err
		}
		if count > 0 {
			return true, fmt.Sprintf("Ya existe un residente registrado con la cédula '%s'", cedula), nil
		}
	}

	if email != "" {
		query := `SELECT COUNT(*) FROM residentes WHERE residencial_id = $1 AND LOWER(TRIM(email)) = LOWER(TRIM($2)) AND deleted_at IS NULL`
		args := []interface{}{residencialID, email}
		if excludeID > 0 {
			query += " AND id <> $3"
			args = append(args, excludeID)
		}
		var count int
		if err := db.DB.QueryRow(query, args...).Scan(&count); err != nil {
			return false, "", err
		}
		if count > 0 {
			return true, fmt.Sprintf("Ya existe un residente registrado con el correo '%s'", email), nil
		}
	}

	return false, "", nil
}

// RegistrarResidente crea el residente y sincroniza sus casas asignadas
func RegistrarResidente(residente model.Residente) (int64, error) {
	if strings.TrimSpace(residente.Nombre) == "" {
		return 0, fmt.Errorf("el nombre del residente es obligatorio")
	}

	duplicado, motivo, err := VerificarResidenteDuplicado(residente.ResidencialID, residente.Cedula, residente.Email, 0)
	if err != nil {
		return 0, fmt.Errorf("error al verificar duplicidad de residente: %w", err)
	}
	if duplicado {
		return 0, fmt.Errorf("DUPLICATE_RESIDENTE: %s", motivo)
	}

	var primaryCasaID sql.NullInt64
	if len(residente.CasasAsignadasIDs) > 0 {
		primaryCasaID = sql.NullInt64{Int64: residente.CasasAsignadasIDs[0], Valid: true}
	} else if residente.CasaID != nil {
		primaryCasaID = sql.NullInt64{Int64: *residente.CasaID, Valid: true}
	}

	query := `
		INSERT INTO residentes (residencial_id, casa_id, nombre, cedula, telefono, telefono_emergencia, email, es_propietario, activo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		RETURNING id`

	var id int64
	err = db.DB.QueryRow(query,
		residente.ResidencialID,
		primaryCasaID,
		strings.TrimSpace(residente.Nombre),
		strings.TrimSpace(residente.Cedula),
		strings.TrimSpace(residente.Telefono),
		strings.TrimSpace(residente.TelefonoEmergencia),
		strings.TrimSpace(residente.Email),
		residente.EsPropietario,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	// Asignar casas si se enviaron
	if len(residente.CasasAsignadasIDs) > 0 {
		_ = SincronizarCasasDeResidente(residente.ResidencialID, id, residente.CasasAsignadasIDs)
	} else if primaryCasaID.Valid {
		_ = SincronizarCasasDeResidente(residente.ResidencialID, id, []int64{primaryCasaID.Int64})
	}

	return id, nil
}

// ActualizarResidente modifica los datos del residente y sincroniza sus casas asignadas
func ActualizarResidente(residente model.Residente) error {
	if residente.ID <= 0 {
		return fmt.Errorf("ID de residente inválido")
	}

	duplicado, motivo, err := VerificarResidenteDuplicado(residente.ResidencialID, residente.Cedula, residente.Email, residente.ID)
	if err != nil {
		return fmt.Errorf("error al verificar duplicidad: %w", err)
	}
	if duplicado {
		return fmt.Errorf("DUPLICATE_RESIDENTE: %s", motivo)
	}

	var primaryCasaID sql.NullInt64
	if len(residente.CasasAsignadasIDs) > 0 {
		primaryCasaID = sql.NullInt64{Int64: residente.CasasAsignadasIDs[0], Valid: true}
	}

	query := `
		UPDATE residentes
		SET nombre = $1, cedula = $2, telefono = $3, telefono_emergencia = $4, email = $5, es_propietario = $6, casa_id = $7
		WHERE id = $8 AND residencial_id = $9 AND deleted_at IS NULL`

	res, err := db.DB.Exec(query,
		strings.TrimSpace(residente.Nombre),
		strings.TrimSpace(residente.Cedula),
		strings.TrimSpace(residente.Telefono),
		strings.TrimSpace(residente.TelefonoEmergencia),
		strings.TrimSpace(residente.Email),
		residente.EsPropietario,
		primaryCasaID,
		residente.ID,
		residente.ResidencialID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("residente no encontrado o eliminado")
	}

	// Sincronizar asignación de casas N:M
	if len(residente.CasasAsignadasIDs) >= 0 {
		_ = SincronizarCasasDeResidente(residente.ResidencialID, residente.ID, residente.CasasAsignadasIDs)
	}

	return nil
}

// SincronizarCasasDeResidente asocia múltiples casas al residente
func SincronizarCasasDeResidente(residencialID string, residenteID int64, casaIDs []int64) error {
	// Eliminar asignaciones anteriores
	_, err := db.DB.Exec("DELETE FROM residente_casas WHERE residencial_id = $1 AND residente_id = $2", residencialID, residenteID)
	if err != nil {
		return err
	}

	for i, cID := range casaIDs {
		isPrincipal := (i == 0)
		_, _ = db.DB.Exec(`
			INSERT INTO residente_casas (residencial_id, residente_id, casa_id, es_principal)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (residencial_id, residente_id, casa_id) DO NOTHING`,
			residencialID, residenteID, cID, isPrincipal)
	}
	return nil
}

// EliminarResidente realiza SoftDelete
func EliminarResidente(id int64, residencialID string) error {
	query := `
		UPDATE residentes
		SET deleted_at = NOW(), activo = false
		WHERE id = $1 AND residencial_id = $2 AND deleted_at IS NULL`

	res, err := db.DB.Exec(query, id, residencialID)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("residente no encontrado o ya eliminado")
	}
	return nil
}
