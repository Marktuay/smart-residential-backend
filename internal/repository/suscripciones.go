package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitPushDB crea la tabla si no existe
func InitPushDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS suscripciones_push (
		id SERIAL PRIMARY KEY,
		residencial_id VARCHAR(50) NOT NULL,
		usuario_id INT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
		endpoint TEXT NOT NULL UNIQUE,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL,
		creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_suscripciones_usuario ON suscripciones_push(usuario_id);
	CREATE INDEX IF NOT EXISTS idx_suscripciones_residencial ON suscripciones_push(residencial_id);
	`
	_, err := db.DB.Exec(query)
	return err
}

// GuardarSuscripcion inserta o actualiza la suscripción push
func GuardarSuscripcion(ctx context.Context, sub *model.SuscripcionPush) error {
	query := `
	INSERT INTO suscripciones_push (residencial_id, usuario_id, endpoint, p256dh, auth)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (endpoint) DO UPDATE 
	SET residencial_id = EXCLUDED.residencial_id,
	    usuario_id = EXCLUDED.usuario_id,
	    p256dh = EXCLUDED.p256dh,
	    auth = EXCLUDED.auth
	`
	_, err := db.DB.ExecContext(ctx, query, sub.ResidencialID, sub.UsuarioID, sub.Endpoint, sub.P256dh, sub.Auth)
	return err
}

// ObtenerSuscripcionesPorUsuario obtiene las suscripciones push de un usuario específico
func ObtenerSuscripcionesPorUsuario(ctx context.Context, usuarioID int64) ([]model.SuscripcionPush, error) {
	query := `
	SELECT id, residencial_id, usuario_id, endpoint, p256dh, auth
	FROM suscripciones_push
	WHERE usuario_id = $1
	`
	rows, err := db.DB.QueryContext(ctx, query, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.SuscripcionPush
	for rows.Next() {
		var sub model.SuscripcionPush
		if err := rows.Scan(&sub.ID, &sub.ResidencialID, &sub.UsuarioID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// ObtenerSuscripcionesPorResidencial obtiene todas las suscripciones de los guardias del residencial
func ObtenerSuscripcionesPorResidencial(ctx context.Context, residencialID string) ([]model.SuscripcionPush, error) {
	query := `
	SELECT s.id, s.residencial_id, s.usuario_id, s.endpoint, s.p256dh, s.auth
	FROM suscripciones_push s
	JOIN usuarios u ON s.usuario_id = u.id
	WHERE s.residencial_id = $1 AND u.rol = 'GUARDIA'
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []model.SuscripcionPush
	for rows.Next() {
		var sub model.SuscripcionPush
		if err := rows.Scan(&sub.ID, &sub.ResidencialID, &sub.UsuarioID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// EliminarSuscripcionPorEndpoint elimina una suscripción si el servidor push devuelve 410 Gone o 404 Not Found
func EliminarSuscripcionPorEndpoint(ctx context.Context, endpoint string) error {
	query := `DELETE FROM suscripciones_push WHERE endpoint = $1`
	_, err := db.DB.ExecContext(ctx, query, endpoint)
	return err
}
