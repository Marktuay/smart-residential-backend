package repository

import (
	"context"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// ObtenerEstadisticasDashboard obtiene todos los conteos para el resumen general
func ObtenerEstadisticasDashboard(ctx context.Context, residencialID string) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}

	// Contar Incidentes Activos (PENDIENTE o EN_PROGRESO)
	err := db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND estado IN ('PENDIENTE', 'EN_PROGRESO')", residencialID).Scan(&stats.Activos)
	if err != nil { return nil, err }

	// Contar Incidentes Críticos (Prioridad ALTA y Activos)
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND prioridad = 'ALTA' AND estado IN ('PENDIENTE', 'EN_PROGRESO')", residencialID).Scan(&stats.Criticos)
	if err != nil { return nil, err }

	// Contar Resueltos Hoy
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND estado = 'RESUELTO' AND DATE(creado_en) = CURRENT_DATE", residencialID).Scan(&stats.ResueltosHoy)
	if err != nil { return nil, err }

	// Total Histórico de Incidentes
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1", residencialID).Scan(&stats.TotalHistorico)
	if err != nil { return nil, err }

	// Contar Rondas Hoy
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM asignaciones_rondas WHERE residencial_id = $1 AND estado = 'FINALIZADA' AND fecha = CURRENT_DATE", residencialID).Scan(&stats.RondasHoy)
	if err != nil { return nil, err }

	// Contar Visitas Hoy
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM visitas WHERE residencial_id = $1 AND estado = 'INGRESÓ' AND DATE(fecha_ingreso) = CURRENT_DATE", residencialID).Scan(&stats.VisitasHoy)
	if err != nil { return nil, err }

	// Complejos (1 en este caso porque estamos dentro de un residencial)
	stats.Complejos = 1

	// Puntos QR Registrados
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM puntos_qr WHERE residencial_id = $1", residencialID).Scan(&stats.CasasQR)
	if err != nil { return nil, err }

	// Usuarios Activos (Guardias/Admin)
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usuarios WHERE residencial_id = $1", residencialID).Scan(&stats.UsuariosActivos)
	if err != nil { return nil, err }

	// Residentes (Para esto podríamos agregar un ROL 'RESIDENTE' y contarlo aparte si fuera necesario, aquí contamos todos)
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usuarios WHERE residencial_id = $1 AND rol = 'RESIDENTE'", residencialID).Scan(&stats.Residentes)
	if err != nil { return nil, err }

	return stats, nil
}
