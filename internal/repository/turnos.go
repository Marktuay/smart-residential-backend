package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
)

// InitTurnosDB crea la tabla y actualiza columnas si no existen
func InitTurnosDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS turnos (
			id SERIAL PRIMARY KEY,
			residencial_id VARCHAR(50) NOT NULL,
			guardia_id INT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
			fecha DATE NOT NULL,
			hora_inicio TIME NOT NULL,
			hora_fin TIME NOT NULL,
			descripcion TEXT,
			ubicacion VARCHAR(150),
			estado VARCHAR(50) NOT NULL DEFAULT 'PROGRAMADO'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_turnos_residencial ON turnos(residencial_id);`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS horas NUMERIC(4,2) DEFAULT 0;`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS asistencia VARCHAR(50) DEFAULT 'PENDIENTE';`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS marcado_entrada TIMESTAMP WITH TIME ZONE NULL;`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS marcado_salida TIMESTAMP WITH TIME ZONE NULL;`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS creado_por VARCHAR(100) DEFAULT '';`,
		`ALTER TABLE turnos ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();`,
		`CREATE INDEX IF NOT EXISTS idx_turnos_fecha_guardia ON turnos(residencial_id, fecha, guardia_id);`,
	}

	for _, q := range queries {
		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error inicializando tabla turnos: %v", err)
		}
	}
	return nil
}

// CalcularHorasTurno calcula la duración exacta del turno soportando turnos que cruzan la medianoche
func CalcularHorasTurno(horaInicio, horaFin string) float64 {
	t1, err1 := time.Parse("15:04", strings.TrimSpace(horaInicio))
	t2, err2 := time.Parse("15:04", strings.TrimSpace(horaFin))
	if err1 != nil || err2 != nil {
		return 0
	}
	diff := t2.Sub(t1).Hours()
	if diff < 0 {
		diff += 24 // Cruza medianoche (ej. 19:00 a 07:00 = 12 horas)
	}
	return diff
}

// VerificarSolapamiento verifica si un colaborador ya tiene un turno activo en ese rango horario
func VerificarSolapamiento(ctx context.Context, residencialID string, guardiaID int64, fecha string, horaInicio, horaFin string, excluirID int64) (bool, string, error) {
	query := `
		SELECT t.id, TO_CHAR(t.hora_inicio, 'HH24:MI'), TO_CHAR(t.hora_fin, 'HH24:MI'), 
		       COALESCE(t.ubicacion, 'Puesto general'), COALESCE(t.descripcion, '')
		FROM turnos t
		WHERE t.residencial_id = $1 
		  AND t.guardia_id = $2 
		  AND t.fecha = $3 
		  AND t.estado != 'CANCELADO'
		  AND t.id != $4
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID, guardiaID, fecha, excluirID)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()

	hIniNuevo, _ := time.Parse("15:04", strings.TrimSpace(horaInicio))
	hFinNuevo, _ := time.Parse("15:04", strings.TrimSpace(horaFin))
	durNueva := hFinNuevo.Sub(hIniNuevo).Minutes()
	if durNueva < 0 {
		durNueva += 24 * 60
	}

	for rows.Next() {
		var id int64
		var exIniStr, exFinStr, ubicacion, desc string
		if err := rows.Scan(&id, &exIniStr, &exFinStr, &ubicacion, &desc); err != nil {
			continue
		}

		hIniEx, _ := time.Parse("15:04", exIniStr)
		hFinEx, _ := time.Parse("15:04", exFinStr)
		durEx := hFinEx.Sub(hIniEx).Minutes()
		if durEx < 0 {
			durEx += 24 * 60
		}

		// Comparación de intervalos dentro de la misma jornada
		// Si se solapa el inicio o fin
		ini1 := hIniNuevo.Hour()*60 + hIniNuevo.Minute()
		fin1 := ini1 + int(durNueva)

		ini2 := hIniEx.Hour()*60 + hIniEx.Minute()
		fin2 := ini2 + int(durEx)

		// Comprobar solapamiento: max(ini1, ini2) < min(fin1, fin2)
		maxIni := ini1
		if ini2 > maxIni {
			maxIni = ini2
		}
		minFin := fin1
		if fin2 < minFin {
			minFin = fin2
		}

		if maxIni < minFin {
			msg := fmt.Sprintf("Conflicto con turno ID %d (%s a %s en %s)", id, exIniStr, exFinStr, ubicacion)
			return true, msg, nil
		}
	}

	return false, "", nil
}

// ObtenerTurnos retorna turnos con datos completos del guardia
func ObtenerTurnos(ctx context.Context, residencialID string) ([]model.Turno, error) {
	query := `
		SELECT t.id, t.residencial_id, t.guardia_id, 
		       COALESCE(u.nombre_completo, u.email, 'Sin Asignar') AS guardia_nombre,
		       COALESCE(u.email, '') AS guardia_email,
		       COALESCE(u.cedula, '') AS guardia_cedula,
		       TO_CHAR(t.fecha, 'YYYY-MM-DD'), 
		       TO_CHAR(t.hora_inicio, 'HH24:MI'), 
		       TO_CHAR(t.hora_fin, 'HH24:MI'), 
		       COALESCE(t.horas, 0),
		       COALESCE(t.descripcion, ''), 
		       COALESCE(t.ubicacion, ''), 
		       t.estado,
		       COALESCE(t.asistencia, 'PENDIENTE'),
		       t.marcado_entrada,
		       t.marcado_salida,
		       COALESCE(t.creado_por, ''),
		       t.created_at
		FROM turnos t
		LEFT JOIN usuarios u ON t.guardia_id = u.id
		WHERE t.residencial_id = $1
		ORDER BY t.fecha DESC, t.hora_inicio ASC
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turnos []model.Turno
	for rows.Next() {
		var t model.Turno
		err := rows.Scan(
			&t.ID, &t.ResidencialID, &t.GuardiaID,
			&t.GuardiaNombre, &t.GuardiaEmail, &t.GuardiaCedula,
			&t.Fecha, &t.HoraInicio, &t.HoraFin, &t.Horas,
			&t.Descripcion, &t.Ubicacion, &t.Estado,
			&t.Asistencia, &t.MarcadoEntrada, &t.MarcadoSalida,
			&t.CreadoPor, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if t.Horas == 0 {
			t.Horas = CalcularHorasTurno(t.HoraInicio, t.HoraFin)
		}
		turnos = append(turnos, t)
	}

	if turnos == nil {
		turnos = []model.Turno{}
	}

	return turnos, nil
}

// CrearTurno inserta un nuevo turno validando anti-solapamiento y cálculo de horas
func CrearTurno(ctx context.Context, t *model.Turno, actor, ip string) error {
	horas := CalcularHorasTurno(t.HoraInicio, t.HoraFin)
	if horas <= 0 {
		return fmt.Errorf("la hora de inicio y fin no pueden ser idénticas")
	}
	if horas > 18 {
		return fmt.Errorf("la duración del turno no puede superar las 18 horas continuas")
	}

	// Validar solapamiento horario
	haySolapamiento, motivo, err := VerificarSolapamiento(ctx, t.ResidencialID, t.GuardiaID, t.Fecha, t.HoraInicio, t.HoraFin, 0)
	if err != nil {
		return fmt.Errorf("error validando horario: %v", err)
	}
	if haySolapamiento {
		return fmt.Errorf("solapamiento detectado: %s", motivo)
	}

	t.Horas = horas
	if t.Estado == "" {
		t.Estado = "PROGRAMADO"
	}
	if t.Asistencia == "" {
		t.Asistencia = "PENDIENTE"
	}
	if t.CreadoPor == "" {
		t.CreadoPor = actor
	}

	query := `
		INSERT INTO turnos (residencial_id, guardia_id, fecha, hora_inicio, hora_fin, horas, descripcion, ubicacion, estado, asistencia, creado_por, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		RETURNING id, created_at
	`
	err = db.DB.QueryRowContext(ctx, query,
		t.ResidencialID,
		t.GuardiaID,
		t.Fecha,
		t.HoraInicio,
		t.HoraFin,
		t.Horas,
		t.Descripcion,
		t.Ubicacion,
		t.Estado,
		t.Asistencia,
		t.CreadoPor,
	).Scan(&t.ID, &t.CreatedAt)

	if err == nil {
		detalles := fmt.Sprintf("Turno creado (ID: %d) para guardia ID: %d en fecha: %s (%s - %s, %.1fh en %s)",
			t.ID, t.GuardiaID, t.Fecha, t.HoraInicio, t.HoraFin, t.Horas, t.Ubicacion)
		go RegistrarAuditoriaAsync(t.ResidencialID, "CREAR_TURNO", actor, ip, detalles)
	}

	return err
}

// ActualizarTurno actualiza un turno existente con validación
func ActualizarTurno(ctx context.Context, t *model.Turno, actor, ip string) error {
	horas := CalcularHorasTurno(t.HoraInicio, t.HoraFin)
	if horas <= 0 {
		return fmt.Errorf("la hora de inicio y fin no pueden ser idénticas")
	}

	haySolapamiento, motivo, err := VerificarSolapamiento(ctx, t.ResidencialID, t.GuardiaID, t.Fecha, t.HoraInicio, t.HoraFin, t.ID)
	if err != nil {
		return fmt.Errorf("error validando horario: %v", err)
	}
	if haySolapamiento {
		return fmt.Errorf("solapamiento detectado: %s", motivo)
	}

	t.Horas = horas
	query := `
		UPDATE turnos
		SET guardia_id = $1, fecha = $2, hora_inicio = $3, hora_fin = $4, horas = $5, descripcion = $6, ubicacion = $7, estado = $8
		WHERE id = $9 AND residencial_id = $10
	`
	_, err = db.DB.ExecContext(ctx, query,
		t.GuardiaID,
		t.Fecha,
		t.HoraInicio,
		t.HoraFin,
		t.Horas,
		t.Descripcion,
		t.Ubicacion,
		t.Estado,
		t.ID,
		t.ResidencialID,
	)

	if err == nil {
		detalles := fmt.Sprintf("Turno ID %d actualizado para guardia ID %d en fecha %s (%s - %s)",
			t.ID, t.GuardiaID, t.Fecha, t.HoraInicio, t.HoraFin)
		go RegistrarAuditoriaAsync(t.ResidencialID, "ACTUALIZAR_TURNO", actor, ip, detalles)
	}

	return err
}

// MarcarAsistenciaTurno registra entrada, salida o estado de asistencia de un turno
func MarcarAsistenciaTurno(ctx context.Context, id int64, residencialID string, asistencia string, actor, ip string) error {
	asistencia = strings.ToUpper(strings.TrimSpace(asistencia))
	var query string
	now := time.Now()

	switch asistencia {
	case "PRESENTE":
		query = `UPDATE turnos SET asistencia = $1, estado = 'EN_CURSO', marcado_entrada = $2 WHERE id = $3 AND residencial_id = $4`
		_, err := db.DB.ExecContext(ctx, query, asistencia, now, id, residencialID)
		if err == nil {
			go RegistrarAuditoriaAsync(residencialID, "MARCAR_ASISTENCIA", actor, ip, fmt.Sprintf("Turno ID %d marcado como PRESENTE", id))
		}
		return err
	case "FINALIZADO", "COMPLETADO":
		query = `UPDATE turnos SET estado = 'COMPLETADO', marcado_salida = $1 WHERE id = $2 AND residencial_id = $3`
		_, err := db.DB.ExecContext(ctx, query, now, id, residencialID)
		if err == nil {
			go RegistrarAuditoriaAsync(residencialID, "MARCAR_ASISTENCIA", actor, ip, fmt.Sprintf("Turno ID %d marcado como FINALIZADO", id))
		}
		return err
	default:
		query = `UPDATE turnos SET asistencia = $1 WHERE id = $2 AND residencial_id = $3`
		_, err := db.DB.ExecContext(ctx, query, asistencia, id, residencialID)
		if err == nil {
			go RegistrarAuditoriaAsync(residencialID, "MARCAR_ASISTENCIA", actor, ip, fmt.Sprintf("Turno ID %d asistencia actualizada a %s", id, asistencia))
		}
		return err
	}
}

// EliminarTurno cancela un turno
func EliminarTurno(ctx context.Context, id int64, residencialID string, actor, ip string) error {
	query := `
		UPDATE turnos
		SET estado = 'CANCELADO'
		WHERE id = $1 AND residencial_id = $2
	`
	_, err := db.DB.ExecContext(ctx, query, id, residencialID)

	if err == nil {
		detalles := fmt.Sprintf("Turno ID %d cancelado", id)
		go RegistrarAuditoriaAsync(residencialID, "ELIMINAR_TURNO", actor, ip, detalles)
	}

	return err
}

// ProgramarTurnosMasivos inserta turnos en lote validando conflictos por cada celda
func ProgramarTurnosMasivos(ctx context.Context, residencialID string, req model.ProgramacionMasivaRequest, actor, ip string) (*model.ProgramacionMasivaResponse, error) {
	dDesde, err := time.Parse("2006-01-02", req.FechaDesde)
	if err != nil {
		return nil, fmt.Errorf("fecha_desde inválida (formato YYYY-MM-DD requerido)")
	}
	dHasta, err := time.Parse("2006-01-02", req.FechaHasta)
	if err != nil {
		return nil, fmt.Errorf("fecha_hasta inválida (formato YYYY-MM-DD requerido)")
	}

	if dHasta.Before(dDesde) {
		return nil, fmt.Errorf("fecha_hasta debe ser igual o posterior a fecha_desde")
	}

	durHoras := CalcularHorasTurno(req.HoraInicio, req.HoraFin)
	if durHoras <= 0 {
		return nil, fmt.Errorf("horas de inicio y fin inválidas")
	}

	resp := &model.ProgramacionMasivaResponse{
		Creados:    0,
		Conflictos: 0,
		Mensajes:   []string{},
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO turnos (residencial_id, guardia_id, fecha, hora_inicio, hora_fin, horas, descripcion, ubicacion, estado, asistencia, creado_por, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PROGRAMADO', 'PENDIENTE', $9, NOW())
	`)
	if err != nil {
		return nil, err
	}
	defer insertStmt.Close()

	for d := dDesde; !d.After(dHasta); d = d.AddDate(0, 0, 1) {
		fechaStr := d.Format("2006-01-02")

		for _, gID := range req.GuardiaIDs {
			// Validar solapamiento
			haySolapamiento, motivo, err := VerificarSolapamiento(ctx, residencialID, gID, fechaStr, req.HoraInicio, req.HoraFin, 0)
			if err != nil {
				resp.Conflictos++
				resp.Mensajes = append(resp.Mensajes, fmt.Sprintf("Guardia ID %d en %s: error validando (%v)", gID, fechaStr, err))
				continue
			}
			if haySolapamiento {
				resp.Conflictos++
				resp.Mensajes = append(resp.Mensajes, fmt.Sprintf("Guardia ID %d en %s: %s", gID, fechaStr, motivo))
				continue
			}

			desc := strings.TrimSpace(req.TipoTurno + " - " + req.Notas)
			_, err = insertStmt.ExecContext(ctx, residencialID, gID, fechaStr, req.HoraInicio, req.HoraFin, durHoras, desc, req.Ubicacion, actor)
			if err != nil {
				resp.Conflictos++
				resp.Mensajes = append(resp.Mensajes, fmt.Sprintf("Error al insertar guardia %d en %s: %v", gID, fechaStr, err))
			} else {
				resp.Creados++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	detalles := fmt.Sprintf("Programación masiva ejecutada: %d turnos creados, %d conflictos omitidos (del %s al %s)",
		resp.Creados, resp.Conflictos, req.FechaDesde, req.FechaHasta)
	go RegistrarAuditoriaAsync(residencialID, "PROGRAMACION_MASIVA_TURNOS", actor, ip, detalles)

	return resp, nil
}

// ListarAuditoriaTurnos consulta las bitácoras específicas de turnos
func ListarAuditoriaTurnos(ctx context.Context, residencialID string) ([]model.AuditLog, error) {
	query := `
		SELECT id, residencial_id, accion, actor, COALESCE(ip, ''), detalles, created_at
		FROM logs_auditoria_sistema
		WHERE residencial_id = $1 
		  AND accion IN ('CREAR_TURNO', 'ACTUALIZAR_TURNO', 'ELIMINAR_TURNO', 'MARCAR_ASISTENCIA', 'PROGRAMACION_MASIVA_TURNOS')
		ORDER BY created_at DESC
		LIMIT 100
	`
	rows, err := db.DB.QueryContext(ctx, query, residencialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		if err := rows.Scan(&l.ID, &l.ResidencialID, &l.Accion, &l.Actor, &l.IP, &l.Detalles, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}

	return logs, nil
}
