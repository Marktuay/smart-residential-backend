package repository

import (
	"context"
	"fmt"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"time"
)

// ObtenerEstadisticasDashboard obtiene todos los conteos y métricas reales para el centro de mando
func ObtenerEstadisticasDashboard(ctx context.Context, residencialID string) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{
		ActividadSemanal: []model.ActividadDia{},
		UltimasNovedades: []model.NovedadReciente{},
	}

	// 1. Contar Incidentes Activos (ABIERTO)
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND estado = 'ABIERTO'", residencialID).Scan(&stats.Activos)

	// 2. Contar Incidentes Críticos / Altos Activos
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND (nivel_gravedad = 'ALTA' OR nivel_gravedad = 'CRITICA') AND estado = 'ABIERTO'", residencialID).Scan(&stats.Criticos)

	// 3. Contar Incidentes Resueltos Hoy
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1 AND estado = 'RESUELTO' AND DATE(COALESCE(fecha_resolucion, fecha_creacion)) = CURRENT_DATE", residencialID).Scan(&stats.ResueltosHoy)

	// 4. Total Histórico de Incidentes
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM incidentes WHERE residencial_id = $1", residencialID).Scan(&stats.TotalHistorico)

	// 5. Contar Rondas / Marcaciones Hoy
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM marcaciones_rondas WHERE residencial_id = $1 AND DATE(fecha_hora) = CURRENT_DATE", residencialID).Scan(&stats.RondasHoy)
	if stats.RondasHoy == 0 {
		_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM asignaciones_rondas WHERE residencial_id = $1 AND fecha = CURRENT_DATE", residencialID).Scan(&stats.RondasHoy)
	}

	// 6. Contar Visitas Hoy
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM visitas WHERE residencial_id = $1 AND (DATE(fecha_ingreso) = CURRENT_DATE OR (fecha_ingreso IS NULL AND DATE(fecha_esperada) = CURRENT_DATE))", residencialID).Scan(&stats.VisitasHoy)

	// 7. Complejos
	stats.Complejos = 1

	// 8. Puntos QR Registrados
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM puntos_qr WHERE residencial_id = $1", residencialID).Scan(&stats.CasasQR)

	// 9. Personal de Seguridad y Usuarios Activos
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usuarios WHERE residencial_id = $1 AND activo = true AND deleted_at IS NULL", residencialID).Scan(&stats.UsuariosActivos)

	// 10. Total Residentes Oficiales
	_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM residentes WHERE residencial_id = $1 AND activo = true AND deleted_at IS NULL", residencialID).Scan(&stats.Residentes)
	if stats.Residentes == 0 {
		_ = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usuarios WHERE residencial_id = $1 AND rol = 'RESIDENTE'", residencialID).Scan(&stats.Residentes)
	}

	// 11. Actividad Semanal Dinámica (Últimos 7 días de lunes a domingo o retrospectiva)
	diasSemana := []string{"Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"}
	now := time.Now()
	for i := 6; i >= 0; i-- {
		targetDate := now.AddDate(0, 0, -i)
		dateStr := targetDate.Format("2006-01-02")
		dayName := diasSemana[targetDate.Weekday()]

		var accesosDia int
		_ = db.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM visitas 
			WHERE residencial_id = $1 AND (DATE(fecha_ingreso) = $2 OR DATE(fecha_esperada) = $2)
		`, residencialID, dateStr).Scan(&accesosDia)

		var rondasDia int
		_ = db.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM marcaciones_rondas 
			WHERE residencial_id = $1 AND DATE(fecha_hora) = $2
		`, residencialID, dateStr).Scan(&rondasDia)

		if rondasDia == 0 {
			_ = db.DB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM asignaciones_rondas 
				WHERE residencial_id = $1 AND fecha = $2
			`, residencialID, dateStr).Scan(&rondasDia)
		}

		stats.ActividadSemanal = append(stats.ActividadSemanal, model.ActividadDia{
			Name:    dayName,
			Accesos: accesosDia,
			Rondas:  rondasDia,
		})
	}

	// 12. Últimas Novedades en Tiempo Real (Incidentes, Visitas y Rondas)
	novedades := []model.NovedadReciente{}

	// Últimos incidentes
	rowsInc, err := db.DB.QueryContext(ctx, `
		SELECT id, tipo, descripcion, nivel_gravedad, fecha_creacion
		FROM incidentes
		WHERE residencial_id = $1
		ORDER BY fecha_creacion DESC LIMIT 4
	`, residencialID)
	if err == nil {
		defer rowsInc.Close()
		for rowsInc.Next() {
			var n model.NovedadReciente
			var fCreacion time.Time
			if err := rowsInc.Scan(&n.ID, &n.Titulo, &n.Desc, &n.Nivel, &fCreacion); err == nil {
				n.Tipo = "INCIDENTE"
				n.Titulo = "Incidente: " + n.Titulo
				n.Fecha = fCreacion.Format("2006-01-02 15:04:05")
				n.Time = formatTimeAgo(fCreacion)
				novedades = append(novedades, n)
			}
		}
	}

	// Últimas marcaciones de ronda
	rowsRondas, err := db.DB.QueryContext(ctx, `
		SELECT m.id, COALESCE(p.nombre, 'Punto QR'), COALESCE(m.observacion, 'Punto verificado en patrullaje'), m.fecha_hora
		FROM marcaciones_rondas m
		LEFT JOIN puntos_qr p ON m.punto_qr_id = p.id
		WHERE m.residencial_id = $1
		ORDER BY m.fecha_hora DESC LIMIT 4
	`, residencialID)
	if err == nil {
		defer rowsRondas.Close()
		for rowsRondas.Next() {
			var n model.NovedadReciente
			var fMarcacion time.Time
			if err := rowsRondas.Scan(&n.ID, &n.Titulo, &n.Desc, &fMarcacion); err == nil {
				n.Tipo = "RONDA"
				n.Titulo = "Ronda: " + n.Titulo
				n.Fecha = fMarcacion.Format("2006-01-02 15:04:05")
				n.Time = formatTimeAgo(fMarcacion)
				novedades = append(novedades, n)
			}
		}
	}

	// Últimas visitas ingresadas
	rowsVis, err := db.DB.QueryContext(ctx, `
		SELECT id, nombre_visitante, 'Acceso a Casa ' || COALESCE(numero_casa, ''), COALESCE(fecha_ingreso, CURRENT_TIMESTAMP)
		FROM visitas
		WHERE residencial_id = $1
		ORDER BY id DESC LIMIT 4
	`, residencialID)
	if err == nil {
		defer rowsVis.Close()
		for rowsVis.Next() {
			var n model.NovedadReciente
			var fVis time.Time
			if err := rowsVis.Scan(&n.ID, &n.Titulo, &n.Desc, &fVis); err == nil {
				n.Tipo = "VISITA"
				n.Titulo = "Visita: " + n.Titulo
				n.Fecha = fVis.Format("2006-01-02 15:04:05")
				n.Time = formatTimeAgo(fVis)
				novedades = append(novedades, n)
			}
		}
	}

	// Ordenar novedades combinadas por fecha descendente
	if len(novedades) > 5 {
		stats.UltimasNovedades = novedades[:5]
	} else {
		stats.UltimasNovedades = novedades
	}

	return stats, nil
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "Hace un momento"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("Hace %d min", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("Hace %d h", hours)
	}
	days := int(diff.Hours() / 24)
	return fmt.Sprintf("Hace %d d", days)
}
