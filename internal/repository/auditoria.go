package repository

import (
	"context"
	"log"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"time"
)

// RegistrarAuditoriaAsync guarda logs de forma asíncrona para no bloquear peticiones HTTP
func RegistrarAuditoriaAsync(residencialID, accion, actor, ip, detalles string) {
	_, err := db.DB.Exec("INSERT INTO logs_auditoria_sistema (residencial_id, accion, actor, ip, detalles, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		residencialID, accion, actor, ip, detalles, time.Now())
	if err != nil {
		log.Printf("Error registrando auditoría: %v", err)
	}
}

// ListarLogs obtiene el log de auditoría de un residencial
func ListarLogs(ctx context.Context, residencialID string) ([]model.AuditLog, error) {
	query := `SELECT id, accion, actor, ip, detalles, created_at 
			  FROM logs_auditoria_sistema WHERE residencial_id = $1 ORDER BY created_at DESC LIMIT 100`

	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		if err := rows.Scan(&l.ID, &l.Accion, &l.Actor, &l.IP, &l.Detalles, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.ResidencialID = residencialID
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}
	return logs, nil
}
