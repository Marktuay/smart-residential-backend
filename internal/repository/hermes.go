package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// InitHermesDB crea las tablas de configuración, alertas y la cuenta de servicio de Hermes
func InitHermesDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS hermes_config (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL UNIQUE,
			activo BOOLEAN NOT NULL DEFAULT true,
			intervalo_evaluacion_seg INT NOT NULL DEFAULT 60,
			max_inactividad_gps_mins INT NOT NULL DEFAULT 5,
			alerta_sos_activa BOOLEAN NOT NULL DEFAULT true,
			alerta_rondas_retraso BOOLEAN NOT NULL DEFAULT true,
			alerta_flota_desvios BOOLEAN NOT NULL DEFAULT true,
			webhook_notificacion TEXT DEFAULT '',
			gemini_api_key TEXT DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`ALTER TABLE hermes_config ADD COLUMN IF NOT EXISTS gemini_api_key TEXT DEFAULT '';`,
		`CREATE INDEX IF NOT EXISTS idx_hermes_config_res ON hermes_config(residencial_id);`,
		`CREATE TABLE IF NOT EXISTS hermes_alertas (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			tipo_evento VARCHAR(50) NOT NULL,
			severidad VARCHAR(20) NOT NULL,
			titulo VARCHAR(200) NOT NULL,
			descripcion TEXT NOT NULL,
			metadata_json JSONB DEFAULT '{}'::jsonb,
			estado VARCHAR(30) NOT NULL DEFAULT 'PENDIENTE',
			atendido_por VARCHAR(100) DEFAULT '',
			fecha_creacion TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			fecha_resolucion TIMESTAMP WITH TIME ZONE NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_hermes_alertas_res_estado ON hermes_alertas(residencial_id, estado);`,
		`CREATE INDEX IF NOT EXISTS idx_hermes_alertas_fecha ON hermes_alertas(fecha_creacion DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_hermes_alertas_tipo ON hermes_alertas(tipo_evento, estado);`,
	}

	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error inicializando tablas hermes: %v", err)
		}
	}

	// Insertar configuración por defecto GLOBAL si no existe
	_, _ = db.DB.Exec(`
		INSERT INTO hermes_config (residencial_id, activo, intervalo_evaluacion_seg, max_inactividad_gps_mins, alerta_sos_activa, alerta_rondas_retraso, alerta_flota_desvios)
		VALUES ('GLOBAL', true, 60, 5, true, true, true)
		ON CONFLICT (residencial_id) DO NOTHING;
	`)

	// Garantizar cuenta de servicio de Hermes como SISADMIN para auditoría y operaciones (no interactiva)
	emailHermes := "hermes.agent@ncs365.com"
	var existeID int64
	err := db.DB.QueryRow("SELECT id FROM usuarios WHERE email = $1", emailHermes).Scan(&existeID)
	if err == sql.ErrNoRows {
		randomBytes := make([]byte, 32)
		if _, errRand := rand.Read(randomBytes); errRand != nil {
			return fmt.Errorf("error generando secreto de servicio de Hermes: %w", errRand)
		}
		passHash, errHash := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(randomBytes)), bcrypt.DefaultCost)
		if errHash != nil {
			return fmt.Errorf("error generando hash de servicio de Hermes: %w", errHash)
		}
		_, errIns := db.DB.Exec(`
			INSERT INTO usuarios (residencial_id, email, nombre_completo, telefono, cedula, password_hash, rol, activo, creado_en)
			VALUES ('GLOBAL', $1, 'Hermes Autonomous Sentinel', '+505 0000-0000', 'HERMES-AI-001', $2, 'SISADMIN', false, NOW())
			ON CONFLICT (email) DO NOTHING;
		`, emailHermes, string(passHash))
		if errIns != nil {
			log.Printf("Advertencia al crear cuenta de servicio de Hermes: %v", errIns)
		} else {
			log.Println("✅ Cuenta de servicio de Hermes Sentinel inicializada (no interactiva).")
		}
	}

	return nil
}

// ObtenerHermesConfig devuelve la configuración del residencial o el default GLOBAL
func ObtenerHermesConfig(ctx context.Context, residencialID string) (*model.HermesConfig, error) {
	query := `
		SELECT id, residencial_id, activo, intervalo_evaluacion_seg, max_inactividad_gps_mins,
		       alerta_sos_activa, alerta_rondas_retraso, alerta_flota_desvios, 
		       COALESCE(webhook_notificacion, ''), COALESCE(gemini_api_key, ''),
		       created_at, updated_at
		FROM hermes_config
		WHERE residencial_id = $1
	`
	var cfg model.HermesConfig
	err := db.DB.QueryRowContext(ctx, query, residencialID).Scan(
		&cfg.ID, &cfg.ResidencialID, &cfg.Activo, &cfg.IntervaloEvaluacionSeg, &cfg.MaxInactividadGpsMins,
		&cfg.AlertaSosActiva, &cfg.AlertaRondasRetraso, &cfg.AlertaFlotaDesvios, 
		&cfg.WebhookNotificacion, &cfg.GeminiAPIKey,
		&cfg.CreatedAt, &cfg.UpdatedAt,
	)

	if err == sql.ErrNoRows && residencialID != "GLOBAL" {
		// Fallback a configuración GLOBAL
		return ObtenerHermesConfig(ctx, "GLOBAL")
	} else if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		// Default seguro si no hay nada en base de datos
		return &model.HermesConfig{
			ResidencialID:          residencialID,
			Activo:                 true,
			IntervaloEvaluacionSeg: 60,
			MaxInactividadGpsMins:  5,
			AlertaSosActiva:        true,
			AlertaRondasRetraso:    true,
			AlertaFlotaDesvios:     true,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
		}, nil
	}

	return &cfg, nil
}

// GuardarHermesConfig actualiza o inserta la configuración de supervisión de un residencial
func GuardarHermesConfig(ctx context.Context, cfg *model.HermesConfig) error {
	query := `
		INSERT INTO hermes_config (
			residencial_id, activo, intervalo_evaluacion_seg, max_inactividad_gps_mins,
			alerta_sos_activa, alerta_rondas_retraso, alerta_flota_desvios, webhook_notificacion, gemini_api_key, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (residencial_id) DO UPDATE SET
			activo = EXCLUDED.activo,
			intervalo_evaluacion_seg = EXCLUDED.intervalo_evaluacion_seg,
			max_inactividad_gps_mins = EXCLUDED.max_inactividad_gps_mins,
			alerta_sos_activa = EXCLUDED.alerta_sos_activa,
			alerta_rondas_retraso = EXCLUDED.alerta_rondas_retraso,
			alerta_flota_desvios = EXCLUDED.alerta_flota_desvios,
			webhook_notificacion = EXCLUDED.webhook_notificacion,
			gemini_api_key = EXCLUDED.gemini_api_key,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return db.DB.QueryRowContext(ctx, query,
		cfg.ResidencialID, cfg.Activo, cfg.IntervaloEvaluacionSeg, cfg.MaxInactividadGpsMins,
		cfg.AlertaSosActiva, cfg.AlertaRondasRetraso, cfg.AlertaFlotaDesvios, 
		cfg.WebhookNotificacion, cfg.GeminiAPIKey,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
}

// CrearHermesAlerta registra una nueva anomalía evitando duplicados recientes del mismo evento (15 minutos)
func CrearHermesAlerta(ctx context.Context, a *model.HermesAlerta) (bool, error) {
	// Deduplicación: Si ya hay una alerta idéntica PENDIENTE en los últimos 15 min, omitir
	var existe int64
	dedupQuery := `
		SELECT id FROM hermes_alertas
		WHERE residencial_id = $1 
		  AND tipo_evento = $2 
		  AND titulo = $3 
		  AND estado = 'PENDIENTE'
		  AND fecha_creacion >= NOW() - INTERVAL '15 minutes'
		LIMIT 1
	`
	err := db.DB.QueryRowContext(ctx, dedupQuery, a.ResidencialID, a.TipoEvento, a.Titulo).Scan(&existe)
	if err == nil {
		// Ya existe una alerta activa reciente; omitimos duplicado
		return false, nil
	}

	if a.MetadataJSON == "" {
		a.MetadataJSON = "{}"
	}
	if a.Estado == "" {
		a.Estado = "PENDIENTE"
	}
	a.FechaCreacion = time.Now()

	insertQuery := `
		INSERT INTO hermes_alertas (
			residencial_id, tipo_evento, severidad, titulo, descripcion, metadata_json, estado, fecha_creacion
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		RETURNING id
	`
	err = db.DB.QueryRowContext(ctx, insertQuery,
		a.ResidencialID, a.TipoEvento, a.Severidad, a.Titulo, a.Descripcion, a.MetadataJSON, a.Estado, a.FechaCreacion,
	).Scan(&a.ID)

	if err != nil {
		return false, err
	}

	// Trazabilidad y auditoría asíncrona
	go RegistrarAuditoriaAsync(a.ResidencialID, "HERMES_ALERTA_GENERADA", "HERMES_CONTROLLER", "127.0.0.1",
		fmt.Sprintf("[%s] %s: %s", a.Severidad, a.TipoEvento, a.Titulo))

	return true, nil
}

// ListarHermesAlertas obtiene las alertas filtradas por residencial, estado y severidad
func ListarHermesAlertas(ctx context.Context, residencialID, estado, severidad string, limit int) ([]model.HermesAlerta, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var conditions []string
	var args []interface{}
	idx := 1

	if residencialID != "" && residencialID != "GLOBAL" && residencialID != "TODOS" {
		conditions = append(conditions, fmt.Sprintf("residencial_id = $%d", idx))
		args = append(args, residencialID)
		idx++
	}

	if estado != "" && estado != "TODOS" {
		conditions = append(conditions, fmt.Sprintf("estado = $%d", idx))
		args = append(args, strings.ToUpper(estado))
		idx++
	}

	if severidad != "" && severidad != "TODAS" {
		conditions = append(conditions, fmt.Sprintf("severidad = $%d", idx))
		args = append(args, strings.ToUpper(severidad))
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, residencial_id, tipo_evento, severidad, titulo, descripcion,
		       metadata_json::text, estado, COALESCE(atendido_por, ''), fecha_creacion, fecha_resolucion
		FROM hermes_alertas
		%s
		ORDER BY 
			CASE severidad 
				WHEN 'CRITICA' THEN 1 
				WHEN 'ALTA' THEN 2 
				WHEN 'MEDIA' THEN 3 
				ELSE 4 
			END ASC,
			fecha_creacion DESC
		LIMIT $%d
	`, whereClause, idx)
	args = append(args, limit)

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alertas []model.HermesAlerta
	for rows.Next() {
		var a model.HermesAlerta
		if err := rows.Scan(
			&a.ID, &a.ResidencialID, &a.TipoEvento, &a.Severidad, &a.Titulo, &a.Descripcion,
			&a.MetadataJSON, &a.Estado, &a.AtendidoPor, &a.FechaCreacion, &a.FechaResolucion,
		); err != nil {
			return nil, err
		}
		alertas = append(alertas, a)
	}

	if alertas == nil {
		alertas = []model.HermesAlerta{}
	}
	return alertas, nil
}

// ResolverHermesAlerta marca una alerta como ATENDIDO o DESCARTADO
func ResolverHermesAlerta(ctx context.Context, id int64, residencialID, nuevoEstado, atendidoPor string) error {
	var query string
	var args []interface{}

	if residencialID == "GLOBAL" || residencialID == "" {
		query = `
			UPDATE hermes_alertas
			SET estado = $1, atendido_por = $2, fecha_resolucion = NOW()
			WHERE id = $3
		`
		args = []interface{}{nuevoEstado, atendidoPor, id}
	} else {
		query = `
			UPDATE hermes_alertas
			SET estado = $1, atendido_por = $2, fecha_resolucion = NOW()
			WHERE id = $3 AND residencial_id = $4
		`
		args = []interface{}{nuevoEstado, atendidoPor, id, residencialID}
	}

	res, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("alerta no encontrada o sin cambios")
	}

	go RegistrarAuditoriaAsync(residencialID, "HERMES_ALERTA_RESUELTA", atendidoPor, "127.0.0.1",
		fmt.Sprintf("Alerta #%d marcada como %s", id, nuevoEstado))

	return nil
}

// ObtenerMetricasHermes devuelve el conteo de alertas pendientes y críticas
func ObtenerMetricasHermes(ctx context.Context, residencialID string) (pendientes int, criticas int, err error) {
	var query string
	var args []interface{}

	if residencialID == "" || residencialID == "GLOBAL" || residencialID == "TODOS" {
		query = `
			SELECT 
				COUNT(*) FILTER (WHERE estado = 'PENDIENTE'),
				COUNT(*) FILTER (WHERE estado = 'PENDIENTE' AND severidad = 'CRITICA')
			FROM hermes_alertas
		`
	} else {
		query = `
			SELECT 
				COUNT(*) FILTER (WHERE estado = 'PENDIENTE'),
				COUNT(*) FILTER (WHERE estado = 'PENDIENTE' AND severidad = 'CRITICA')
			FROM hermes_alertas
			WHERE residencial_id = $1
		`
		args = append(args, residencialID)
	}

	err = db.DB.QueryRowContext(ctx, query, args...).Scan(&pendientes, &criticas)
	return pendientes, criticas, err
}

// ContarResidencialesMonitoreados devuelve la cantidad de residenciales activos
func ContarResidencialesMonitoreados(ctx context.Context) (int, error) {
	var count int
	err := db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM residenciales").Scan(&count)
	return count, err
}
