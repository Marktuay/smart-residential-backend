package hermes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// Engine es la instancia del controlador autónomo Hermes
type Engine struct {
	mu           sync.RWMutex
	db           *sql.DB
	ctx          context.Context
	cancel       context.CancelFunc
	startTime    time.Time
	lastRun      time.Time
	isRunning    bool
	interval     time.Duration
	version      string
}

var (
	defaultEngine *Engine
	once          sync.Once
)

// GetEngine devuelve la instancia singleton de Hermes
func GetEngine() *Engine {
	once.Do(func() {
		defaultEngine = &Engine{
			db:        db.DB,
			startTime: time.Now(),
			interval:  60 * time.Second,
			version:   "v1.0.0-production",
		}
	})
	return defaultEngine
}

// Start arranca el ciclo centinela de Hermes en segundo plano
func (e *Engine) Start(ctx context.Context, interval time.Duration) {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return
	}
	if interval > 0 {
		e.interval = interval
	}
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.isRunning = true
	e.startTime = time.Now()
	e.mu.Unlock()

	log.Printf("🤖 Hermes Sentinel Engine iniciado (Frecuencia de supervisión: %v)", e.interval)

	// Ejecutar primera pasada inmediatamente en goroutine
	go func() {
		e.runCycle(e.ctx)
		ticker := time.NewTicker(e.interval)
		defer ticker.Stop()

		for {
			select {
			case <-e.ctx.Done():
				log.Println("🛑 Hermes Sentinel Engine detenido.")
				return
			case <-ticker.C:
				e.runCycle(e.ctx)
			}
		}
	}()
}

// Stop detiene el ciclo de ejecución
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.isRunning {
		return
	}
	if e.cancel != nil {
		e.cancel()
	}
	e.isRunning = false
}

// TriggerEvaluation ejecuta una ronda de evaluación de inmediato
func (e *Engine) TriggerEvaluation(ctx context.Context) error {
	e.runCycle(ctx)
	return nil
}

// GetStatus recopila el estado actual y métricas de salud
func (e *Engine) GetStatus(ctx context.Context, residencialID string) model.HermesStatusResponse {
	e.mu.RLock()
	uptime := int64(time.Since(e.startTime).Seconds())
	lastRun := e.lastRun
	isRunning := e.isRunning
	version := e.version
	e.mu.RUnlock()

	pendientes, criticas, _ := repository.ObtenerMetricasHermes(ctx, residencialID)
	monitoreados, _ := repository.ContarResidencialesMonitoreados(ctx)

	mensaje := "Hermes operando normalmente. Todos los perímetros bajo supervisión."
	if criticas > 0 {
		mensaje = fmt.Sprintf("¡ATENCIÓN! Hay %d alerta(s) crítica(s) activa(s) que requieren intervención inmediata.", criticas)
	} else if pendientes > 0 {
		mensaje = fmt.Sprintf("Hermes activo. Hay %d alerta(s) operativa(s) en seguimiento.", pendientes)
	}

	return model.HermesStatusResponse{
		Activo:                    isRunning,
		Version:                   version,
		UltimaEvaluacion:          lastRun,
		TotalAlertasPendientes:    pendientes,
		TotalAlertasCriticas:      criticas,
		ResidencialesMonitoreados: monitoreados,
		UptimeSegundos:            uptime,
		Mensaje:                   mensaje,
	}
}

// runCycle ejecuta todas las reglas de control de Hermes
func (e *Engine) runCycle(ctx context.Context) {
	if db.DB == nil {
		return // Sin base de datos activa (entorno de pruebas o pre-inicialización)
	}

	e.mu.Lock()
	e.lastRun = time.Now()
	e.mu.Unlock()

	// 1. Evaluar Inactividad GPS en Guardias con Turno Activo
	if err := e.evaluarInactividadGPS(ctx); err != nil {
		log.Printf("⚠️ [Hermes] Error evaluando inactividad GPS: %v", err)
	}

	// 2. Evaluar Incidentes Críticos y SOS Activos
	if err := e.evaluarIncidentesCriticos(ctx); err != nil {
		log.Printf("⚠️ [Hermes] Error evaluando incidentes críticos: %v", err)
	}

	// 3. Evaluar Rondas con Retraso o Incompletas
	if err := e.evaluarRondasRetraso(ctx); err != nil {
		log.Printf("⚠️ [Hermes] Error evaluando rondas con retraso: %v", err)
	}

	// 4. Evaluar Anomalías en Flota 360
	if err := e.evaluarAnomaliasFlota(ctx); err != nil {
		log.Printf("⚠️ [Hermes] Error evaluando anomalías de flota: %v", err)
	}
}

// 1. Inactividad GPS de guardias en turno
func (e *Engine) evaluarInactividadGPS(ctx context.Context) error {
	// Buscar turnos vigentes hoy cuya ventana horaria esté en curso
	query := `
		SELECT 
			t.id, t.residencial_id, t.guardia_id, 
			COALESCE(u.nombre_completo, u.email) AS guardia_nombre,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - r.actualizado_en))/60, 999) AS minutos_inactivo,
			COALESCE(hc.max_inactividad_gps_mins, 5) AS max_mins
		FROM turnos t
		JOIN usuarios u ON u.id = t.guardia_id
		LEFT JOIN rastreo_ubicacion r ON r.usuario_id = t.guardia_id
		LEFT JOIN hermes_config hc ON hc.residencial_id = t.residencial_id
		WHERE t.fecha = CURRENT_DATE
		  AND t.estado != 'CANCELADO'
		  AND (t.asistencia = 'PRESENTE' OR (CURRENT_TIME BETWEEN t.hora_inicio AND t.hora_fin))
	`
	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var turnoID, guardiaID int64
		var residencialID, guardiaNombre string
		var minutosInactivo float64
		var maxMins int

		if err := rows.Scan(&turnoID, &residencialID, &guardiaID, &guardiaNombre, &minutosInactivo, &maxMins); err != nil {
			continue
		}

		if int(minutosInactivo) >= maxMins {
			meta, _ := json.Marshal(map[string]interface{}{
				"turno_id":         turnoID,
				"guardia_id":       guardiaID,
				"minutos_inactivo": int(minutosInactivo),
				"limite_mins":      maxMins,
			})

			alerta := &model.HermesAlerta{
				ResidencialID: residencialID,
				TipoEvento:    "INACTIVIDAD_GPS",
				Severidad:     "ALTA",
				Titulo:        fmt.Sprintf("Pérdida de señal GPS: %s", guardiaNombre),
				Descripcion:   fmt.Sprintf("El colaborador %s no transmite ubicación hace %d minutos durante su turno activo.", guardiaNombre, int(minutosInactivo)),
				MetadataJSON:  string(meta),
			}

			if creada, _ := repository.CrearHermesAlerta(ctx, alerta); creada {
				log.Printf("🚨 [Hermes] Alerta generada: Inactividad GPS guardia %s (%s)", guardiaNombre, residencialID)
			}
		}
	}

	return nil
}

// 2. Incidentes Críticos o SOS no atendidos
func (e *Engine) evaluarIncidentesCriticos(ctx context.Context) error {
	query := `
		SELECT 
			id, residencial_id, tipo, descripcion, nivel_gravedad, 
			reportado_por, COALESCE(ubicacion, ''), fecha_creacion
		FROM incidentes
		WHERE estado = 'ABIERTO'
		  AND (nivel_gravedad = 'CRITICA' OR tipo IN ('SOS', 'EMERGENCIA', 'PANICO', 'BOTON_PANICO'))
		  AND fecha_creacion >= NOW() - INTERVAL '4 hours'
	`
	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var incID int64
		var residencialID, tipo, desc, gravedad, reportadoPor, ubicacion string
		var fechaCreacion time.Time

		if err := rows.Scan(&incID, &residencialID, &tipo, &desc, &gravedad, &reportadoPor, &ubicacion, &fechaCreacion); err != nil {
			continue
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"incidente_id":   incID,
			"reportado_por":  reportadoPor,
			"ubicacion":      ubicacion,
			"nivel_gravedad": gravedad,
		})

		alerta := &model.HermesAlerta{
			ResidencialID: residencialID,
			TipoEvento:    "BOTON_PANICO_SOS",
			Severidad:     "CRITICA",
			Titulo:        fmt.Sprintf("Alerta SOS / Emergencia: %s", tipo),
			Descripcion:   fmt.Sprintf("Incidente crítico reportado por %s en %s: %s", reportadoPor, ubicacion, desc),
			MetadataJSON:  string(meta),
		}

		if creada, _ := repository.CrearHermesAlerta(ctx, alerta); creada {
			log.Printf("🔥 [Hermes] Alerta CRÍTICA generada: %s en %s", tipo, residencialID)
		}
	}

	return nil
}

// 3. Rondas perimetrales con retraso o incompletas
func (e *Engine) evaluarRondasRetraso(ctx context.Context) error {
	query := `
		SELECT 
			ar.id, ar.residencial_id, pr.nombre AS programa_nombre,
			COALESCE(u.nombre_completo, u.email) AS guardia_nombre,
			COALESCE(ar.puntos_completados, 0),
			COALESCE((SELECT COUNT(*) FROM programas_puntos pp WHERE pp.programa_id = ar.programa_id), 0) AS puntos_total
		FROM asignaciones_rondas ar
		JOIN programas_rondas pr ON pr.id = ar.programa_id
		LEFT JOIN usuarios u ON u.id = ar.usuario_id
		WHERE ar.fecha::date = CURRENT_DATE
		  AND ar.estado NOT IN ('COMPLETADA', 'FINALIZADA')
		  AND CURRENT_TIME > (pr.hora_fin::time + INTERVAL '15 minutes')
	`
	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var asignacionID int64
		var residencialID, programaNombre, guardiaNombre string
		var completados, total int

		if err := rows.Scan(&asignacionID, &residencialID, &programaNombre, &guardiaNombre, &completados, &total); err != nil {
			continue
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"asignacion_ronda_id": asignacionID,
			"puntos_completados":  completados,
			"puntos_total":        total,
			"guardia":             guardiaNombre,
		})

		alerta := &model.HermesAlerta{
			ResidencialID: residencialID,
			TipoEvento:    "RONDA_VENCIDA",
			Severidad:     "MEDIA",
			Titulo:        fmt.Sprintf("Ronda incompleta fuera de horario: %s", programaNombre),
			Descripcion:   fmt.Sprintf("La ronda asignada a %s finalizó con %d/%d puntos registrados.", guardiaNombre, completados, total),
			MetadataJSON:  string(meta),
		}

		if creada, _ := repository.CrearHermesAlerta(ctx, alerta); creada {
			log.Printf("⏳ [Hermes] Ronda vencida detectada: %s (%s)", programaNombre, residencialID)
		}
	}

	return nil
}

// 4. Anomalías en Flota 360
func (e *Engine) evaluarAnomaliasFlota(ctx context.Context) error {
	query := `
		SELECT 
			sf.id, sf.residencial_id, sf.estado_actual, COALESCE(fv.placa, 'S/P') AS vehiculo_placa
		FROM solicitudes_flota sf
		LEFT JOIN flota_vehiculos fv ON fv.id = sf.vehiculo_id
		WHERE sf.estado_actual IN ('INSPECCION_TALLER_PENDIENTE', 'GARITA_SALIDA_PENDIENTE')
		  AND sf.creado_en < NOW() - INTERVAL '3 hours'
	`
	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var solID int64
		var residencialID, estado, placa string

		if err := rows.Scan(&solID, &residencialID, &estado, &placa); err != nil {
			continue
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"solicitud_id": solID,
			"estado":       estado,
			"placa":        placa,
		})

		alerta := &model.HermesAlerta{
			ResidencialID: residencialID,
			TipoEvento:    "FLOTA_ANOMALIA",
			Severidad:     "MEDIA",
			Titulo:        fmt.Sprintf("Retención de Flota 360: Unidad %s", placa),
			Descripcion:   fmt.Sprintf("La solicitud #%d con vehículo %s excede 3 horas en estado %s sin completar.", solID, placa, estado),
			MetadataJSON:  string(meta),
		}

		if creada, _ := repository.CrearHermesAlerta(ctx, alerta); creada {
			log.Printf("🚜 [Hermes] Anomalía de flota detectada: Solicitud #%d", solID)
		}
	}

	return nil
}
