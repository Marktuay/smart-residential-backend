package hermes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ProcessChatMessage procesa una consulta en lenguaje natural usando Gemini o el motor analítico integral de Hermes
func ProcessChatMessage(ctx context.Context, residencialID, mensaje string) (*model.HermesChatResponse, error) {
	cleanMsg := strings.TrimSpace(mensaje)
	if cleanMsg == "" {
		return nil, fmt.Errorf("el mensaje no puede estar vacío")
	}

	// 1. Recopilar Contexto Operativo Integral en Tiempo Real
	contexto, err := recopilarContextoOperativo(ctx, residencialID)
	if err != nil {
		contexto = map[string]interface{}{"error_contexto": err.Error()}
	}

	// 2. Determinar si disponemos de Gemini API Key (variable de entorno o configuración en BD)
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		cfg, _ := repository.ObtenerHermesConfig(ctx, residencialID)
		if cfg != nil && cfg.GeminiAPIKey != "" {
			apiKey = strings.TrimSpace(cfg.GeminiAPIKey)
		}
	}

	// Si hay API Key válida, consultar a Gemini (con fallback automático de modelo)
	if apiKey != "" {
		respGemini, err := consultarGemini(ctx, apiKey, cleanMsg, contexto, residencialID)
		if err == nil && respGemini != "" {
			return &model.HermesChatResponse{
				Respuesta: respGemini,
				Fuente:    "GEMINI_AI",
				Fecha:     time.Now().Format("2006-01-02 15:04:05"),
				Contexto:  contexto,
			}, nil
		}
	}

	// 3. Motor Analítico Autónomo Integral (Funciona 100% offline y sin costos de API)
	respAnalitica := generarRespuestaAnalitica(cleanMsg, contexto, residencialID)

	return &model.HermesChatResponse{
		Respuesta: respAnalitica,
		Fuente:    "HERMES_ANALYTICS",
		Fecha:     time.Now().Format("2006-01-02 15:04:05"),
		Contexto:  contexto,
	}, nil
}

// recopilarContextoOperativo extrae telemetría y datos de todos los subsistemas en tiempo real
func recopilarContextoOperativo(ctx context.Context, residencialID string) (map[string]interface{}, error) {
	ctxMap := make(map[string]interface{})
	ctxMap["residencial_id"] = residencialID
	ctxMap["timestamp_utc"] = time.Now().UTC().Format(time.RFC3339)

	// A. Guardias activos transmitiendo GPS en los ÚLTIMOS 15 MINUTOS (Telemetría viva)
	queryGps := `
		SELECT u.id, 
		       COALESCE(NULLIF(u.nombre_completo, ''), u.email, 'Guardia #' || u.id::text) AS nombre,
		       r.latitud, r.longitud, 
		       ROUND(EXTRACT(EPOCH FROM (NOW() - r.actualizado_en))/60) AS mins_ago,
		       COALESCE(r.velocidad, 0) AS velocidad,
		       COALESCE(r.precision, 0) AS precision_m
		FROM rastreo_ubicacion r
		JOIN usuarios u ON u.id = r.usuario_id
		WHERE ($1 = 'GLOBAL' OR r.residencial_id = $1)
		  AND r.actualizado_en >= NOW() - INTERVAL '15 minutes'
		ORDER BY r.actualizado_en DESC
	`
	rowsGps, err := db.DB.QueryContext(ctx, queryGps, residencialID)
	var guardiasGPS []map[string]interface{}
	if err == nil {
		defer rowsGps.Close()
		for rowsGps.Next() {
			var id int64
			var nombre string
			var lat, lng, vel, prec float64
			var minsAgo int
			if err := rowsGps.Scan(&id, &nombre, &lat, &lng, &minsAgo, &vel, &prec); err == nil {
				guardiasGPS = append(guardiasGPS, map[string]interface{}{
					"id": id, "nombre": nombre, "lat": lat, "lng": lng, 
					"hace_minutos": minsAgo, "velocidad": vel, "precision": prec,
				})
			}
		}
	}
	ctxMap["guardias_con_gps_activo"] = len(guardiasGPS)
	ctxMap["detalle_guardias_gps"] = guardiasGPS

	// A.2 Catálogo de personal de seguridad registrado en el residencial
	queryPersonal := `
		SELECT u.id, u.email, COALESCE(NULLIF(u.nombre_completo, ''), u.email) AS nombre, 
		       u.rol, u.activo,
		       COALESCE(r.latitud, 0) AS ultima_lat,
		       COALESCE(r.longitud, 0) AS ultima_lng,
		       ROUND(EXTRACT(EPOCH FROM (NOW() - r.actualizado_en))/60) AS ultima_baliza_mins
		FROM usuarios u
		LEFT JOIN rastreo_ubicacion r ON r.usuario_id = u.id
		WHERE ($1 = 'GLOBAL' OR u.residencial_id = $1)
		  AND (u.rol LIKE '%GUARDIA%' OR u.rol = 'SUPERVISOR')
		  AND u.activo = true AND u.deleted_at IS NULL
		ORDER BY u.id ASC
	`
	rowsPers, err := db.DB.QueryContext(ctx, queryPersonal, residencialID)
	var catalogoGuardias []map[string]interface{}
	if err == nil {
		defer rowsPers.Close()
		for rowsPers.Next() {
			var id int64
			var email, nombre, rol string
			var activo bool
			var ulat, ulng float64
			var minsAgo sql.NullInt64
			if err := rowsPers.Scan(&id, &email, &nombre, &rol, &activo, &ulat, &ulng, &minsAgo); err == nil {
				item := map[string]interface{}{
					"id": id, "email": email, "nombre": nombre, "rol": rol, "activo": activo,
					"lat": ulat, "lng": ulng,
				}
				if minsAgo.Valid {
					item["ultima_baliza_mins"] = minsAgo.Int64
				} else {
					item["ultima_baliza_mins"] = int64(-1)
				}
				catalogoGuardias = append(catalogoGuardias, item)
			}
		}
	}
	ctxMap["total_guardias_plantilla"] = len(catalogoGuardias)
	ctxMap["catalogo_guardias"] = catalogoGuardias

	// B. Turnos del día de hoy
	queryTurnos := `
		SELECT t.id, 
		       COALESCE(NULLIF(u.nombre_completo, ''), u.email, 'Guardia #' || t.guardia_id::text) AS nombre,
		       t.hora_inicio, t.hora_fin, 
		       COALESCE(t.asistencia, 'PENDIENTE') AS asistencia, 
		       t.estado
		FROM turnos t
		JOIN usuarios u ON u.id = t.guardia_id
		WHERE t.fecha = CURRENT_DATE 
		  AND ($1 = 'GLOBAL' OR t.residencial_id = $1)
		  AND t.estado != 'CANCELADO'
		ORDER BY t.hora_inicio ASC
	`
	rowsTurnos, err := db.DB.QueryContext(ctx, queryTurnos, residencialID)
	var turnosHoy []map[string]interface{}
	var presentesCount int
	if err == nil {
		defer rowsTurnos.Close()
		for rowsTurnos.Next() {
			var id int64
			var nombre, inicio, fin, asistencia, estado string
			if err := rowsTurnos.Scan(&id, &nombre, &inicio, &fin, &asistencia, &estado); err == nil {
				if asistencia == "PRESENTE" {
					presentesCount++
				}
				turnosHoy = append(turnosHoy, map[string]interface{}{
					"id": id, "guardia": nombre, "horario": fmt.Sprintf("%s - %s", inicio, fin),
					"asistencia": asistencia, "estado": estado,
				})
			}
		}
	}
	ctxMap["total_turnos_hoy"] = len(turnosHoy)
	ctxMap["guardias_presentes"] = presentesCount
	ctxMap["detalle_turnos"] = turnosHoy

	// C. Rondas perimetrales asignadas para hoy
	queryRondas := `
		SELECT ar.id, 
		       COALESCE(pr.nombre, 'Ronda General') AS programa_nombre,
		       COALESCE(NULLIF(u.nombre_completo, ''), u.email, 'Guardia #' || ar.guardia_id::text) AS guardia,
		       COALESCE(pr.hora_inicio, '00:00') AS hora_inicio, 
		       COALESCE(pr.hora_fin, '23:59') AS hora_fin,
		       COALESCE(ar.puntos_completados, 0) AS puntos_completados,
		       COALESCE((SELECT COUNT(*) FROM programas_puntos pp WHERE pp.programa_id = pr.id), 0) AS puntos_total,
		       ar.estado
		FROM asignaciones_rondas ar
		JOIN programas_rondas pr ON pr.id = ar.programa_id
		JOIN usuarios u ON u.id = ar.guardia_id
		WHERE ar.fecha::date = CURRENT_DATE
		  AND ($1 = 'GLOBAL' OR ar.residencial_id = $1)
		ORDER BY pr.hora_inicio ASC
	`
	rowsRondas, err := db.DB.QueryContext(ctx, queryRondas, residencialID)
	var rondasHoy []map[string]interface{}
	var rondasCompletadas int
	if err == nil {
		defer rowsRondas.Close()
		for rowsRondas.Next() {
			var id int64
			var progNombre, guardia, inicio, fin, estado string
			var ptsComp, ptsTot int
			if err := rowsRondas.Scan(&id, &progNombre, &guardia, &inicio, &fin, &ptsComp, &ptsTot, &estado); err == nil {
				if ptsTot > 0 && ptsComp >= ptsTot {
					rondasCompletadas++
				}
				rondasHoy = append(rondasHoy, map[string]interface{}{
					"id": id, "programa": progNombre, "guardia": guardia,
					"horario": fmt.Sprintf("%s - %s", inicio, fin),
					"puntos_completados": ptsComp, "puntos_total": ptsTot,
					"estado": estado,
				})
			}
		}
	}
	ctxMap["total_rondas_hoy"] = len(rondasHoy)
	ctxMap["rondas_completadas"] = rondasCompletadas
	ctxMap["detalle_rondas"] = rondasHoy

	// Total de marcaciones de rondas registradas hoy
	var marcacionesHoy int
	_ = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM marcaciones_rondas 
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1) AND DATE(fecha_hora) = CURRENT_DATE
	`, residencialID).Scan(&marcacionesHoy)
	ctxMap["total_marcaciones_hoy"] = marcacionesHoy

	// D. Flota Pesada y Maquinaria Amarilla 360
	queryFlotaVehiculos := `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE estado = 'OPERATIVO'),
		       COUNT(*) FILTER (WHERE estado = 'EN_TALLER' OR estado = 'MANTENIMIENTO')
		FROM flota_vehiculos
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1)
	`
	var totalVehiculos, vehiculosOperativos, vehiculosEnTaller int
	_ = db.DB.QueryRowContext(ctx, queryFlotaVehiculos, residencialID).Scan(&totalVehiculos, &vehiculosOperativos, &vehiculosEnTaller)
	ctxMap["flota_total_vehiculos"] = totalVehiculos
	ctxMap["flota_operativos"] = vehiculosOperativos
	ctxMap["flota_en_taller"] = vehiculosEnTaller

	querySolicitudesFlota := `
		SELECT sf.id, sf.codigo_solicitud, COALESCE(fv.placa, 'S/P'), sf.punto_origen_id, sf.punto_destino_id, sf.estado_actual
		FROM solicitudes_flota sf
		LEFT JOIN flota_vehiculos fv ON fv.id = sf.vehiculo_id
		WHERE ($1 = 'GLOBAL' OR sf.residencial_id = $1)
		  AND sf.estado_actual NOT IN ('CERRADO_FINALIZADO', 'CANCELADA')
		ORDER BY sf.creado_en DESC LIMIT 5
	`
	rowsFlota, err := db.DB.QueryContext(ctx, querySolicitudesFlota, residencialID)
	var flotaActiva []map[string]interface{}
	if err == nil {
		defer rowsFlota.Close()
		for rowsFlota.Next() {
			var id int64
			var cod, placa, origen, destino, estado string
			if err := rowsFlota.Scan(&id, &cod, &placa, &origen, &destino, &estado); err == nil {
				flotaActiva = append(flotaActiva, map[string]interface{}{
					"id": id, "codigo": cod, "placa": placa,
					"ruta": fmt.Sprintf("%s ➔ %s", origen, destino),
					"estado": estado,
				})
			}
		}
	}
	ctxMap["flota_solicitudes_activas"] = len(flotaActiva)
	ctxMap["detalle_flota_activa"] = flotaActiva

	// E. Garita de Control y Visitas del día
	var visitasHoy, visitasEnCurso int
	_ = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE estado IN ('EN_CURSO', 'INGRESO_AUTORIZADO', 'ACTIVO'))
		FROM visitas
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1)
		  AND (DATE(fecha_ingreso) = CURRENT_DATE OR (fecha_ingreso IS NULL AND DATE(fecha_esperada) = CURRENT_DATE))
	`, residencialID).Scan(&visitasHoy, &visitasEnCurso)
	ctxMap["total_visitas_hoy"] = visitasHoy
	ctxMap["visitas_en_curso"] = visitasEnCurso

	queryUltimasVisitas := `
		SELECT id, nombre_visitante, COALESCE(NULLIF(placa_vehiculo, ''), 'PEATONAL'), 
		       COALESCE(numero_casa, 'S/C'), estado,
		       TO_CHAR(COALESCE(fecha_ingreso, fecha_esperada), 'HH24:MI')
		FROM visitas
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1)
		  AND (DATE(fecha_ingreso) = CURRENT_DATE OR (fecha_ingreso IS NULL AND DATE(fecha_esperada) = CURRENT_DATE))
		ORDER BY COALESCE(fecha_ingreso, fecha_creacion) DESC LIMIT 5
	`
	rowsVisitas, err := db.DB.QueryContext(ctx, queryUltimasVisitas, residencialID)
	var ultimasVisitas []map[string]interface{}
	if err == nil {
		defer rowsVisitas.Close()
		for rowsVisitas.Next() {
			var id int64
			var visitante, placa, casa, estado, hora string
			if err := rowsVisitas.Scan(&id, &visitante, &placa, &casa, &estado, &hora); err == nil {
				ultimasVisitas = append(ultimasVisitas, map[string]interface{}{
					"id": id, "visitante": visitante, "placa": placa, "casa": casa,
					"estado": estado, "hora": hora,
				})
			}
		}
	}
	ctxMap["detalle_ultimas_visitas"] = ultimasVisitas

	// F. Casas, Residentes y Perímetro
	var totalCasas, casasConContrato int
	_ = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE tiene_contrato_seguridad = true)
		FROM casas
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1) AND activo = true AND deleted_at IS NULL
	`, residencialID).Scan(&totalCasas, &casasConContrato)
	ctxMap["total_casas"] = totalCasas
	ctxMap["casas_con_contrato"] = casasConContrato

	var puntosQRTotal int
	_ = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM puntos_qr WHERE ($1 = 'GLOBAL' OR residencial_id = $1)
	`, residencialID).Scan(&puntosQRTotal)
	ctxMap["total_puntos_qr"] = puntosQRTotal

	// G. Incidentes Críticos o Abiertos en últimas 24 horas
	queryInc := `
		SELECT id, tipo, nivel_gravedad, estado, COALESCE(ubicacion, 'S/U'), descripcion, reportado_por,
		       TO_CHAR(fecha_creacion, 'HH24:MI')
		FROM incidentes
		WHERE ($1 = 'GLOBAL' OR residencial_id = $1)
		  AND fecha_creacion >= NOW() - INTERVAL '24 hours'
		ORDER BY fecha_creacion DESC LIMIT 8
	`
	rowsInc, err := db.DB.QueryContext(ctx, queryInc, residencialID)
	var incidentes []map[string]interface{}
	var criticosCount int
	if err == nil {
		defer rowsInc.Close()
		for rowsInc.Next() {
			var id int64
			var tipo, grav, estado, ubi, desc, rep, hora string
			if err := rowsInc.Scan(&id, &tipo, &grav, &estado, &ubi, &desc, &rep, &hora); err == nil {
				if grav == "CRITICA" || tipo == "SOS" {
					criticosCount++
				}
				incidentes = append(incidentes, map[string]interface{}{
					"id": id, "tipo": tipo, "severidad": grav, "estado": estado,
					"ubicacion": ubi, "reportado_por": rep, "descripcion": desc, "hora": hora,
				})
			}
		}
	}
	ctxMap["incidentes_ultimas_24h"] = len(incidentes)
	ctxMap["incidentes_criticos_count"] = criticosCount
	ctxMap["detalle_incidentes"] = incidentes

	// H. Alertas de Hermes Sentinel Activas/Pendientes
	alertas, _ := repository.ListarHermesAlertas(ctx, residencialID, "PENDIENTE", "TODAS", 10)
	ctxMap["alertas_hermes_pendientes"] = len(alertas)
	var alertasList []map[string]interface{}
	for _, a := range alertas {
		alertasList = append(alertasList, map[string]interface{}{
			"id": a.ID, "severidad": a.Severidad, "tipo": a.TipoEvento, 
			"titulo": a.Titulo, "descripcion": a.Descripcion,
		})
	}
	ctxMap["detalle_alertas_hermes"] = alertasList

	return ctxMap, nil
}

// consultarGemini invoca modelos de Google Generative Language API con fallback automático
func consultarGemini(ctx context.Context, apiKey, pregunta string, contexto map[string]interface{}, residencialID string) (string, error) {
	ctxJSON, _ := json.MarshalIndent(contexto, "", "  ")

	systemInstruction := fmt.Sprintf(`
Eres Hermes Sentinel, el agente controlador, administrador autónomo de seguridad y guía oficial del sistema New Century Security (NCS365).
Tu misión doble es:
A) Asistir ejecutivamente con telemetría real a directores, administradores y supervisores de seguridad.
B) Servir como Guía Interactiva y Manual de Uso paso a paso para cualquier usuario que pregunte cómo usar la plataforma Web o la App Móvil.

Estás a cargo de la instalación: %s.

Aquí tienes la radiografía y telemetría en tiempo real completa del sistema (guardias GPS, rondas, turnos, garita/visitas, flota pesada 360, casas, incidentes y alertas):
%s

=== MANUAL OPERATIVO Y GUÍA DE USO DEL SISTEMA NCS365 ===
1. PLATAFORMA WEB (https://app.ncsecurity.net):
   - Selector Multi-Tenant (Barra Superior): Antes de crear casas, rondas, turnos o puntos QR, el usuario debe verificar qué Residencial tiene seleccionado en el selector superior.
   - Centro de Mando (ruta: /): Resumen General con métricas en vivo del residencial.
   - Mapa Operativo (ruta: /map): Tiene 2 pestañas: "En Vivo" (guardias transmitiendo GPS en tiempo real) e "Historial" (traza la ruta histórica de un guardia por fecha/hora con segmentación inteligente que evita líneas rectas falsas sobre casas).
   - Hermes Sentinel (ruta: /hermes): Bandeja de alertas autónomas, métricas de salud, botón de engranaje (⚙️) para configurar umbrales (ej. max_inactividad_gps_mins) y este Chat IA.
   - Control de Rondas (ruta: /rondas) y Puntos de Control QR (ruta: /puntos-qr): Para configurar rondas, primero se crean los Puntos QR (se pueden asociar opcionalmente a una Casa para mostrar escudo verde si tiene contrato de seguridad activo o rojo si no), luego en /rondas se crea el programa y se asigna a un guardia y horario.
   - Control de Accesos / Garita (ruta: /visitas): Registro de entrada y salida de visitantes/vehículos. Incluye botón para enviar notificación asistida por WhatsApp (wa.me) y Push Web al residente.
   - Gestión de Incidencias (ruta: /incidentes): Bitácora de novedades con severidad (BAJA, MEDIA, ALTA, CRITICA), mapa y carga Drag & Drop de evidencia fotográfica.
   - Gestión Residencial (ruta: /residencial): Alta de nuevos residenciales/condominios, configuración geográfica (Latitud Centro, Longitud Centro y Zoom por defecto para el mapa), y gestión de Casas/Viviendas con el flag "tiene_contrato_seguridad".
   - Programación de Turnos (ruta: /programacion): Matriz GRID mensual por colaborador con turnos TD (Turno Día), TN (Turno Noche), EX (Extra) y modal de "Programación Masiva de Turnos" (con protección antiduplicados). Acceso restringido a ADMIN y SISADMIN.
   - Control de Flota 360 (Módulo 360):
     * /flota (Solicitudes y FSM): Flujo irreversible de 10 estados (DRAFT -> ASIGNADA -> INSPECCION_TALLER_PENDIENTE -> APROBADA_TALLER -> GARITA_SALIDA_PENDIENTE -> EN_TRANSITO -> GARITA_LLEGADA_PENDIENTE -> CHECKIN -> INSPECCION_CIERRE_TALLER -> CERRADO_FINALIZADO).
     * /flota/inspeccion (Checklist 360 PWA): Formulario táctil para tablets IP68 con captura fotográfica y lectura de horómetro/km.
     * /flota/vehiculos: Catálogo de vehículos y maquinaria amarilla.
   - Personal y Usuarios (ruta: /usuarios): Crear/editar usuarios y asignar roles (SISADMIN, ADMIN, SUPERVISOR, GUARDIA, GUARDIA_PATRULLERO, GUARDIA_MOTORIZADO, OPERADOR_C2, RESIDENTE, OPERADOR_OPERACIONES, GESTOR_TRANSPORTE, MECANICO_TALLER, GUARDIA_GARITA).
   - Despachar Alertas (ruta: /notificar): Envío de comunicados/alertas (ADMIN y SISADMIN).
   - Gestión de Cobros (ruta: /cobros): Módulo financiero conectado a API externa de cobros.
   - Auditoría (ruta: /auditoria): Registro inmutable de acciones sensibles (solo SISADMIN).

2. APP MÓVIL ANDROID (NCS365 Guarda):
   - Si inicia sesión un GUARDIA: Abre la pantalla operativa con:
     * Switch "Patrullaje / GPS en Vivo": Activa el Servicio en Primer Plano con notificación persistente (transmite cada 8 metros o cada 20s aunque la pantalla esté apagada). Recomendación en el teléfono: Ubicación en "Permitir todo el tiempo" y Batería en "Sin restricciones".
     * Escáner QR: Para marcar puntos de ronda con cámara y linterna.
     * Garita / Visitas, Reportar Incidente (con GPS automático), Mis Turnos (para marcar asistencia PRESENTE) y Botón Rojo SOS.
   - Si inicia sesión un ADMIN, SISADMIN o SUPERVISOR: Abre automáticamente la consola móvil de Hermes (Alertas en vivo con vibración háptica, Chat IA, Supervisión GPS en vivo, y opción "Modo Guarda" en el menú superior para cubrir un puesto o escanear QR sin cerrar sesión).

Reglas de respuesta:
1. Responde siempre en español, de forma clara, profesional y directa.
2. Usa viñetas claras, negritas en nombres de menús/rutas e íconos temáticos (🛡️, 🚨, 🚶, 🚛, 🚪, 📍, ⏱️, ✅, 📖).
3. Si el usuario pregunta por datos operativos en vivo (guardias, rondas, flota, visitas, incidentes), basa tus números estrictamente en los hechos del contexto en tiempo real anterior; nunca inventes datos.
4. Si el usuario pregunta cómo hacer algo en el sistema, dónde está una opción o pide ayuda de uso, guíalo paso a paso usando el Manual Operativo anterior con las rutas exactas y requisitos previos.
5. Si hay incidentes críticos, botones de pánico SOS o guardias sin señal GPS en turno, ponlo como alerta prioritaria al inicio cuando corresponda.
`, residencialID, string(ctxJSON))

	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemInstruction},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": pregunta},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.3,
			"maxOutputTokens": 2500,
		},
	}

	bodyBytes, _ := json.Marshal(payload)

	// Modelos soportados con fallback automático
	modelos := []string{"gemini-3.5-flash", "gemini-3.5-flash-lite", "gemini-3.1-flash-lite-preview", "gemini-3.8-flash", "gemini-flash-latest"}
	client := &http.Client{Timeout: 15 * time.Second}

	var ultimoError error
	for _, mod := range modelos {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", mod, apiKey)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			ultimoError = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			ultimoError = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ultimoError = fmt.Errorf("error api gemini (%s, status %d): %s", mod, resp.StatusCode, string(respBody))
			continue
		}

		var resData struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal(respBody, &resData); err == nil && len(resData.Candidates) > 0 {
			var sb strings.Builder
			for _, part := range resData.Candidates[0].Content.Parts {
				if part.Text != "" {
					sb.WriteString(part.Text)
				}
			}
			if sb.Len() > 0 {
				return sb.String(), nil
			}
		}
	}

	return "", ultimoError
}

// generarRespuestaAnalitica motor autónomo de inteligencia operacional con soporte exhaustivo multi-intención
func generarRespuestaAnalitica(pregunta string, ctx map[string]interface{}, residencialID string) string {
	p := strings.ToLower(pregunta)

	getInt := func(key string) int {
		if val, ok := ctx[key].(int); ok {
			return val
		}
		return 0
	}

	totalGPS := getInt("guardias_con_gps_activo")
	totalTurnos := getInt("total_turnos_hoy")
	guardiasPresentes := getInt("guardias_presentes")
	criticos := getInt("incidentes_criticos_count")
	incidentes24h := getInt("incidentes_ultimas_24h")
	alertasHermes := getInt("alertas_hermes_pendientes")
	totalRondas := getInt("total_rondas_hoy")
	rondasCompletadas := getInt("rondas_completadas")
	marcacionesHoy := getInt("total_marcaciones_hoy")
	totalFlota := getInt("flota_total_vehiculos")
	flotaOperativa := getInt("flota_operativos")
	flotaTaller := getInt("flota_en_taller")
	flotaActiva := getInt("flota_solicitudes_activas")
	totalVisitas := getInt("total_visitas_hoy")
	visitasEnCurso := getInt("visitas_en_curso")
	totalCasas := getInt("total_casas")
	casasContrato := getInt("casas_con_contrato")
	totalQR := getInt("total_puntos_qr")

	// ==========================================
	// 0. INTENCIÓN: GUÍA DE USO DEL SISTEMA (MODO OFFLINE)
	// ==========================================
	if guiaResp, ok := generarGuiaUsoSistema(p); ok {
		return guiaResp
	}

	// ==========================================
	// 1. INTENCIÓN: SALUDOS / PRESENTACIÓN / AYUDA
	// ==========================================
	if strings.HasPrefix(p, "hola") || strings.Contains(p, "buen") || strings.Contains(p, "buenas") ||
		strings.Contains(p, "saludos") || strings.Contains(p, "quien eres") || strings.Contains(p, "que eres") ||
		strings.Contains(p, "que haces") || strings.Contains(p, "ayuda") || strings.Contains(p, "capacidades") ||
		strings.Contains(p, "comandos") || strings.Contains(p, "menu") || strings.Contains(p, "opciones") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🤖 **Hermes Sentinel | Centro de Mando NCS365 en %s**\n\n", residencialID))
		sb.WriteString("¡Saludos cordiales! Soy el **Agente Autónomo Controlador y Administrador de Seguridad** de la urbanización. Mi función es vigilar perimetralmente las operaciones 24/7 y asistir a administradores y supervisores en tiempo real.\n\n")
		sb.WriteString("**¿En qué puedo asistirte en este momento? Puedes consultarme sobre:**\n")
		sb.WriteString(fmt.Sprintf("• 🛡️ **Personal y Turnos:** `%d programados`, `%d transmitiendo GPS vivo` (\"guardias activos\", \"dónde está un guardia\").\n", totalTurnos, totalGPS))
		sb.WriteString(fmt.Sprintf("• 🚶 **Rondas Perimetrales:** `%d programas hoy`, `%d marcaciones registradas` (\"estado de rondas\", \"puntos QR\").\n", totalRondas, marcacionesHoy))
		sb.WriteString(fmt.Sprintf("• 🚛 **Flota Pesada 360:** `%d vehículos`, `%d solicitudes en tránsito/taller` (\"reporte de flota\", \"maquinaria\").\n", totalFlota, flotaActiva))
		sb.WriteString(fmt.Sprintf("• 🚪 **Garita y Visitas:** `%d ingresos hoy`, `%d dentro del predio` (\"visitas activas\", \"últimos ingresos\").\n", totalVisitas, visitasEnCurso))
		sb.WriteString(fmt.Sprintf("• 🚨 **Incidentes y Alertas:** `%d novedades (24h)`, `%d emergencias críticas` (\"incidentes\", \"alertas centinela\").\n", incidentes24h, criticos))
		sb.WriteString(fmt.Sprintf("• 🏡 **Comunidad y Casas:** `%d viviendas`, `%d con contrato activo` (\"estado de casas\", \"cobertura\").\n\n", totalCasas, casasContrato))
		sb.WriteString("💡 *Escribe tu pregunta directamente o consulta un resumen general diciendo `estado del residencial`.*")
		return sb.String()
	}

	// ==========================================
	// 2. INTENCIÓN: RONDAS Y RECORRIDOS PERIMETRALES
	// ==========================================
	if strings.Contains(p, "ronda") || strings.Contains(p, "recorrido") || strings.Contains(p, "patrull") || 
		strings.Contains(p, "punto qr") || strings.Contains(p, "puntos qr") || strings.Contains(p, "marcacion") || 
		strings.Contains(p, "circuito") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🚶 **Estado de Rondas y Patrullaje Perimetral | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Programas de ronda asignados hoy:** %d\n", totalRondas))
		sb.WriteString(fmt.Sprintf("* **Rondas finalizadas al 100%%:** %d de %d\n", rondasCompletadas, totalRondas))
		sb.WriteString(fmt.Sprintf("* **Puntos QR escaneados hoy:** %d marcaciones\n", marcacionesHoy))
		sb.WriteString(fmt.Sprintf("* **Catálogo de Puntos QR instalados:** %d puntos georreferenciados\n\n", totalQR))

		if detalle, ok := ctx["detalle_rondas"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Detalle de Programas y Rondas de Hoy:**\n")
			for _, r := range detalle {
				estadoIcon := "⏳"
				if r["puntos_total"].(int) > 0 && r["puntos_completados"].(int) >= r["puntos_total"].(int) {
					estadoIcon = "✅"
				} else if r["puntos_completados"].(int) > 0 {
					estadoIcon = "🔄"
				}
				sb.WriteString(fmt.Sprintf("%s **%s** (%s)\n", estadoIcon, r["programa"], r["horario"]))
				sb.WriteString(fmt.Sprintf("   • Guardia asignado: **%s**\n", r["guardia"]))
				sb.WriteString(fmt.Sprintf("   • Avance: **%d de %d puntos completados** (Estado: %s)\n", 
					r["puntos_completados"], r["puntos_total"], r["estado"]))
			}
		} else {
			sb.WriteString("ℹ️ *No hay asignaciones de rondas activas registradas para la fecha de hoy en este residencial.*\n")
		}

		if totalRondas > 0 && rondasCompletadas < totalRondas {
			sb.WriteString("\n⚠️ *Hermes Sentinel supervisa el horario de finalización con tolerancia de 15 minutos para alertar retrasos.*")
		}
		return sb.String()
	}

	// ==========================================
	// 3. INTENCIÓN: FLOTA PESADA, MAQUINARIA Y TALLER 360
	// ==========================================
	if strings.Contains(p, "flota") || strings.Contains(p, "vehiculo") || strings.Contains(p, "vehículo") || 
		strings.Contains(p, "maquinaria") || strings.Contains(p, "camion") || strings.Contains(p, "camión") || 
		strings.Contains(p, "taller") || strings.Contains(p, "checklist") || strings.Contains(p, "inspeccion") || 
		strings.Contains(p, "inspección") || strings.Contains(p, "horometro") || strings.Contains(p, "transporte") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🚛 **Control de Flota Pesada, Maquinaria y Checklist 360 | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Unidades en catálogo:** %d vehículos/maquinaria\n", totalFlota))
		sb.WriteString(fmt.Sprintf("* **Unidades operativas:** %d\n", flotaOperativa))
		sb.WriteString(fmt.Sprintf("* **Unidades en taller / mantenimiento:** %d\n", flotaTaller))
		sb.WriteString(fmt.Sprintf("* **Solicitudes de ruta en tránsito o gestión activa:** %d\n\n", flotaActiva))

		if detalle, ok := ctx["detalle_flota_activa"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Movimientos y Solicitudes en Curso:**\n")
			for _, f := range detalle {
				sb.WriteString(fmt.Sprintf("• Solicitud **#%s** | Unidad: **%s**\n", f["codigo"], f["placa"]))
				sb.WriteString(fmt.Sprintf("   • Ruta: %s\n", f["ruta"]))
				sb.WriteString(fmt.Sprintf("   • Estado actual del ciclo: `%s`\n", f["estado"]))
			}
		} else {
			sb.WriteString("✅ *No se reportan solicitudes pendientes de inspección mecánica o unidades retenidas fuera de norma.*\n")
		}
		return sb.String()
	}

	// ==========================================
	// 4. INTENCIÓN: GARITA DE CONTROL, VISITAS Y ACCESOS
	// ==========================================
	if strings.Contains(p, "visita") || strings.Contains(p, "garita") || strings.Contains(p, "acceso") || 
		strings.Contains(p, "entrada") || strings.Contains(p, "salida") || strings.Contains(p, "porter") || 
		strings.Contains(p, "visitante") || strings.Contains(p, "placa") || strings.Contains(p, "quien entro") || 
		strings.Contains(p, "quien está") || strings.Contains(p, "quien esta") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🚪 **Control de Accesos, Garita y Visitas | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Total de visitas registradas hoy:** %d visitante(s)\n", totalVisitas))
		sb.WriteString(fmt.Sprintf("* **Vehículos / Visitas activas en el predio:** %d en curso\n\n", visitasEnCurso))

		if detalle, ok := ctx["detalle_ultimas_visitas"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Últimos movimientos de Garita:**\n")
			for _, v := range detalle {
				sb.WriteString(fmt.Sprintf("• [%s] **%s** ➔ Casa **%s** (Modo: %s, Estado: `%s`)\n", 
					v["hora"], v["visitante"], v["casa"], v["placa"], v["estado"]))
			}
		} else {
			sb.WriteString("ℹ️ *No hay registros de ingreso de visitantes asentados en la bitácora del día de hoy.*\n")
		}
		return sb.String()
	}

	// ==========================================
	// 5. INTENCIÓN: BÚSQUEDA ESPECÍFICA DE UN GUARDIA POR NOMBRE O CONSULTA DE PLANTILLA
	// ==========================================
	if catalogo, ok := ctx["catalogo_guardias"].([]map[string]interface{}); ok && len(catalogo) > 0 {
		for _, g := range catalogo {
			nombre := strings.ToLower(g["nombre"].(string))
			email := strings.ToLower(g["email"].(string))
			emailPrefix := strings.Split(email, "@")[0]
			nomCorto := strings.Split(nombre, " ")[0]
			idStr := fmt.Sprintf("%v", g["id"])

			coincide := false
			if len(nombre) >= 3 && strings.Contains(p, nombre) {
				coincide = true
			} else if len(nomCorto) >= 3 && strings.Contains(p, nomCorto) {
				coincide = true
			} else if len(emailPrefix) >= 3 && strings.Contains(p, emailPrefix) {
				coincide = true
			} else if strings.Contains(p, idStr) && (strings.Contains(p, "guarda") || strings.Contains(p, "guardia")) {
				coincide = true
			}

			if coincide {
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("👮 **Colaborador Localizado en el Sistema: %s**\n\n", g["nombre"]))
				sb.WriteString(fmt.Sprintf("* **ID de Usuario:** #%v\n", g["id"]))
				sb.WriteString(fmt.Sprintf("* **Rol:** `%s`\n", g["rol"]))
				sb.WriteString(fmt.Sprintf("* **Correo institucional:** `%s`\n", g["email"]))
				sb.WriteString(fmt.Sprintf("* **Residencial Asignado:** %s\n", residencialID))
				sb.WriteString(fmt.Sprintf("* **Estado en Nómina:** %s\n\n", map[bool]string{true: "🟢 ACTIVO", false: "🔴 INACTIVO"}[g["activo"].(bool)]))

				// Verificar si tiene turno hoy
				tieneTurno := false
				if turnos, ok := ctx["detalle_turnos"].([]map[string]interface{}); ok {
					for _, t := range turnos {
						if t["id"] == g["id"] || strings.EqualFold(t["guardia"].(string), g["nombre"].(string)) {
							tieneTurno = true
							sb.WriteString(fmt.Sprintf("* **Turno de Hoy:** Horario `%s` | Asistencia: `%s` (Estado: `%s`)\n", t["horario"], t["asistencia"], t["estado"]))
							break
						}
					}
				}
				if !tieneTurno {
					sb.WriteString("* **Turno de Hoy:** ℹ️ *No tiene turno de vigilancia programado para la fecha de hoy.*\n")
				}

				// Verificar telemetría GPS
				minsAgo := g["ultima_baliza_mins"].(int64)
				lat := g["lat"].(float64)
				lng := g["lng"].(float64)
				if minsAgo >= 0 && minsAgo <= 15 {
					sb.WriteString(fmt.Sprintf("* **Telemetría GPS en Vivo:** 🟢 Transmitiendo baliza activa (hace **%d min**) en `Lat: %.5f, Lng: %.5f`\n", minsAgo, lat, lng))
				} else if minsAgo > 15 {
					dias := minsAgo / 1440
					horas := (minsAgo % 1440) / 60
					tiempoTxt := fmt.Sprintf("hace %d horas", horas)
					if dias > 0 {
						tiempoTxt = fmt.Sprintf("hace %d días", dias)
					}
					sb.WriteString(fmt.Sprintf("* **Telemetría GPS:** 🟡 Última señal registrada %s (`Lat: %.5f, Lng: %.5f`). Actualmente no está emitiendo en vivo.\n", tiempoTxt, lat, lng))
				} else {
					sb.WriteString("* **Telemetría GPS:** ⚪ Sin registros de balizas GPS emitidas desde este usuario.\n")
				}

				return sb.String()
			}
		}
	}

	// ==========================================
	// 6. INTENCIÓN: PERSONAL, GUARDIAS Y GPS EN VIVO
	// ==========================================
	if strings.Contains(p, "guardia") || strings.Contains(p, "personal") || strings.Contains(p, "gps") || 
		strings.Contains(p, "ubicacion") || strings.Contains(p, "ubicación") || strings.Contains(p, "donde estan") || 
		strings.Contains(p, "dónde están") || strings.Contains(p, "baliza") || strings.Contains(p, "rastreo") || 
		strings.Contains(p, "inactivo") || strings.Contains(p, "coordenada") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🛡️ **Personal Operativo y Telemetría GPS en Vivo | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Personal con turno programado hoy:** %d\n", totalTurnos))
		sb.WriteString(fmt.Sprintf("* **Asistencia confirmada (Presentes):** %d de %d\n", guardiasPresentes, totalTurnos))
		sb.WriteString(fmt.Sprintf("* **Guardias transmitiendo GPS activo (<15m):** %d\n\n", totalGPS))

		if detalle, ok := ctx["detalle_guardias_gps"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Balizas Activas en Tiempo Real:**\n")
			for _, g := range detalle {
				nombre := g["nombre"].(string)
				if strings.TrimSpace(nombre) == "" {
					nombre = fmt.Sprintf("Guardia ID #%v", g["id"])
				}
				sb.WriteString(fmt.Sprintf("• 🟢 **%s** — Hace %v min | Coord: `%.4f, %.4f` (Vel: %.1f km/h, Prec: %.1fm)\n", 
					nombre, g["hace_minutos"], g["lat"], g["lng"], g["velocidad"], g["precision"]))
			}
		} else {
			sb.WriteString("⚠️ *Actualmente no hay ningún guardia emitiendo coordenadas GPS vivas en los últimos 15 minutos.*\n")
		}

		// Alerta de discrepancia si hay turnos pero no hay GPS
		if totalTurnos > 0 && totalGPS == 0 {
			sb.WriteString("\n🚨 **Alerta Táctica:** Se registran turnos asignados pero ningún guardia está transmitiendo baliza GPS activa. Se sugiere verificar garita o radios.")
		}
		return sb.String()
	}

	// ==========================================
	// 7. INTENCIÓN: TURNOS, PROGRAMACIÓN Y ASISTENCIA
	// ==========================================
	if strings.Contains(p, "turno") || strings.Contains(p, "programacion") || strings.Contains(p, "programación") || 
		strings.Contains(p, "horario") || strings.Contains(p, "asistencia") || strings.Contains(p, "quien cubre") || 
		strings.Contains(p, "noche") || strings.Contains(p, "dia") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 **Programación de Turnos y Cobertura Operativa | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Turnos totales hoy:** %d asignación(es)\n", totalTurnos))
		sb.WriteString(fmt.Sprintf("* **Check-in / Presentes:** %d\n\n", guardiasPresentes))

		if detalle, ok := ctx["detalle_turnos"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Nómina de Turnos del Día:**\n")
			for _, t := range detalle {
				asistBadge := "⏳ Pendiente"
				if t["asistencia"] == "PRESENTE" {
					asistBadge = "✅ Presente"
				}
				sb.WriteString(fmt.Sprintf("• **%s** (%s) ➔ Asistencia: `%s` | Estado: `%s`\n", 
					t["guardia"], t["horario"], asistBadge, t["estado"]))
			}
		} else {
			sb.WriteString("ℹ️ *No se han planificado turnos en el módulo de Programación para el día de hoy.*\n")
		}
		return sb.String()
	}

	// ==========================================
	// 8. INTENCIÓN: INCIDENTES, EMERGENCIAS Y SOS
	// ==========================================
	if strings.Contains(p, "incidente") || strings.Contains(p, "emergencia") || strings.Contains(p, "sos") || 
		strings.Contains(p, "panico") || strings.Contains(p, "pánico") || strings.Contains(p, "novedad") || 
		strings.Contains(p, "robo") || strings.Contains(p, "alarma") || strings.Contains(p, "gravedad") || 
		strings.Contains(p, "critico") || strings.Contains(p, "crítico") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🚨 **Bitácora de Incidentes y Novedades (Últimas 24h) | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Emergencias Críticas / SOS:** %d\n", criticos))
		sb.WriteString(fmt.Sprintf("* **Incidentes Totales Reportados:** %d\n\n", incidentes24h))

		if detalle, ok := ctx["detalle_incidentes"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Últimas Incidencias Reportadas:**\n")
			for _, inc := range detalle {
				iconoSev := "⚠️"
				if inc["severidad"] == "CRITICA" || inc["tipo"] == "SOS" {
					iconoSev = "🚨"
				}
				sb.WriteString(fmt.Sprintf("%s [%s] **%s** (%s)\n", iconoSev, inc["hora"], inc["tipo"], inc["severidad"]))
				sb.WriteString(fmt.Sprintf("   • Ubicación: **%s** | Reportó: %s\n", inc["ubicacion"], inc["reportado_por"]))
				sb.WriteString(fmt.Sprintf("   • Detalle: *%s* (Estado: `%s`)\n", inc["descripcion"], inc["estado"]))
			}
		} else {
			sb.WriteString("✅ *Perímetro sin novedades graves ni botones de pánico reportados en las últimas 24 horas.*\n")
		}
		return sb.String()
	}

	// ==========================================
	// 9. INTENCIÓN: ALERTAS DE HERMES SENTINEL
	// ==========================================
	if strings.Contains(p, "alerta") || strings.Contains(p, "anomalia") || strings.Contains(p, "anomalía") || 
		strings.Contains(p, "centinela") || strings.Contains(p, "pendiente") || strings.Contains(p, "resolver") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("⚡ **Bandeja de Alertas Autónomas de Hermes Sentinel | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Total de alertas pendientes:** %d\n\n", alertasHermes))

		if detalle, ok := ctx["detalle_alertas_hermes"].([]map[string]interface{}); ok && len(detalle) > 0 {
			sb.WriteString("**Alertas Activas que Requieren Atención:**\n")
			for _, a := range detalle {
				sb.WriteString(fmt.Sprintf("• Alerta **#%v** [%s] **%s**\n", a["id"], a["severidad"], a["titulo"]))
				sb.WriteString(fmt.Sprintf("   • Detalle: %s\n", a["descripcion"]))
			}
			sb.WriteString("\n💡 *Puedes resolver o descartar estas alertas desde la pestaña de Alertas en la consola web o app móvil.*")
		} else {
			sb.WriteString("✅ *Hermes Sentinel no tiene alertas pendientes de resolución en este momento. Operación estable.*\n")
		}
		return sb.String()
	}

	// ==========================================
	// 10. INTENCIÓN: CASAS, VIVIENDAS Y RESIDENTES
	// ==========================================
	if strings.Contains(p, "casa") || strings.Contains(p, "vivienda") || strings.Contains(p, "residente") || 
		strings.Contains(p, "contrato") || strings.Contains(p, "urbanizacion") || strings.Contains(p, "urbanización") {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🏡 **Censo Inmobiliario y Cobertura de Seguridad | %s:**\n\n", residencialID))
		sb.WriteString(fmt.Sprintf("* **Total de Casas registradas:** %d viviendas\n", totalCasas))
		sb.WriteString(fmt.Sprintf("* **Casas con Contrato de Seguridad Activo:** %d\n", casasContrato))
		
		pct := 0.0
		if totalCasas > 0 {
			pct = (float64(casasContrato) / float64(totalCasas)) * 100
		}
		sb.WriteString(fmt.Sprintf("* **Nivel de Cobertura Residencial:** %.1f%%\n", pct))
		sb.WriteString(fmt.Sprintf("* **Puntos QR instalados en viviendas/perímetro:** %d\n", totalQR))
		return sb.String()
	}

	// ==========================================
	// 11. INTENCIÓN: ESTADO GENERAL / RESUMEN EJECUTIVO
	// ==========================================
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 **Diagnóstico Operativo Global de Hermes Sentinel | %s:**\n\n", residencialID))

	nivelRiesgo := "🟢 BAJO / CONTROLADO"
	if criticos > 0 {
		nivelRiesgo = fmt.Sprintf("🔴 ELEVADO (%d emergencia(s) crítica(s))", criticos)
	} else if alertasHermes > 0 {
		nivelRiesgo = fmt.Sprintf("🟡 ATENCIÓN REQUERIDA (%d alerta(s) de centinela)", alertasHermes)
	}
	sb.WriteString(fmt.Sprintf("* **Nivel de Riesgo Global:** %s\n", nivelRiesgo))
	sb.WriteString(fmt.Sprintf("* **Personal & GPS:** %d turno(s) hoy | %d baliza(s) GPS viva(s) en últimos 15 min\n", totalTurnos, totalGPS))
	sb.WriteString(fmt.Sprintf("* **Rondas Perimetrales:** %d programa(s) | %d ronda(s) completada(s) | %d escaneos QR\n", totalRondas, rondasCompletadas, marcacionesHoy))
	sb.WriteString(fmt.Sprintf("* **Garita & Accesos:** %d visita(s) hoy | %d vehículo(s)/visita(s) dentro del predio\n", totalVisitas, visitasEnCurso))
	sb.WriteString(fmt.Sprintf("* **Flota Pesada 360:** %d unidades operativas | %d solicitud(es) de movimiento activa(s)\n", flotaOperativa, flotaActiva))
	sb.WriteString(fmt.Sprintf("* **Comunidad:** %d casas registradas (%d con servicio activo de seguridad)\n\n", totalCasas, casasContrato))

	if alertasHermes > 0 {
		sb.WriteString("⚠️ **Anomalías detectadas por el Centinela:**\n")
		if detalle, ok := ctx["detalle_alertas_hermes"].([]map[string]interface{}); ok {
			for _, a := range detalle {
				sb.WriteString(fmt.Sprintf("• [%s] %s\n", a["severidad"], a["titulo"]))
			}
		}
	} else {
		sb.WriteString("✅ **Perímetros y puestos operativos dentro de los estándares de seguridad de NCS365.**\n")
	}

	sb.WriteString("\n*Puedes preguntarme por:* `guardias activos`, `estado de rondas`, `reporte de flota`, `visitas en garita`, `incidentes`, o `dónde está un guardia`.")
	return sb.String()
}

// generarGuiaUsoSistema proporciona instrucciones paso a paso del sistema NCS365 en modo offline
func generarGuiaUsoSistema(p string) (string, bool) {
	esPreguntaGuia := strings.Contains(p, "como ") || strings.Contains(p, "cómo ") ||
		strings.Contains(p, "donde ") || strings.Contains(p, "dónde ") ||
		strings.Contains(p, "guia") || strings.Contains(p, "guía") ||
		strings.Contains(p, "manual") || strings.Contains(p, "tutorial") ||
		strings.Contains(p, "usar el sistema") || strings.Contains(p, "para que sirve") ||
		strings.Contains(p, "para qué sirve") || strings.Contains(p, "pasos para")

	if !esPreguntaGuia {
		return "", false
	}

	// Evitar interceptar preguntas de telemetría como "¿cómo está el residencial?" o "¿dónde está el guarda?"
	if strings.Contains(p, "como esta") || strings.Contains(p, "cómo está") ||
		strings.Contains(p, "donde esta") || strings.Contains(p, "dónde está") {
		return "", false
	}

	if strings.Contains(p, "turno") || strings.Contains(p, "programacion") || strings.Contains(p, "programación") || strings.Contains(p, "horario") {
		return "📖 **Guía: Programación de Turnos (`/programacion`)**\n\n" +
			"1. Verifica que tienes seleccionado el **Residencial correcto** en la barra superior.\n" +
			"2. En el menú lateral izquierdo, entra a **Gestión de Turnos ➔ Programación** (`/programacion`).\n" +
			"3. Verás la **Matriz GRID mensual** por cada guardia.\n" +
			"4. Haz clic en una celda para asignar o editar un turno (`TD` Turno Día, `TN` Turno Noche, `EX` Extra), o usa el botón **Programación Masiva** para asignar varios días de una vez (incluye validación antiduplicados).", true
	}

	if strings.Contains(p, "ronda") || strings.Contains(p, "qr") {
		return "📖 **Guía: Control de Rondas y Puntos QR (`/puntos-qr` y `/rondas`)**\n\n" +
			"1. Primero entra a **Operaciones ➔ Puntos de Control QR** (`/puntos-qr`) para registrar los puntos físicos del perímetro (puedes asociarlos a una Casa para ver el escudo verde si tiene contrato activo).\n" +
			"2. Luego ve a **Operaciones ➔ Control de Rondas** (`/rondas`) para crear el circuito, seleccionar los puntos QR y asignarlo a un guardia con su horario.\n" +
			"3. El guardia escaneará cada código desde la **App Móvil NCS365 Guarda ➔ Escanear QR**.", true
	}

	if strings.Contains(p, "visita") || strings.Contains(p, "garita") || strings.Contains(p, "acceso") || strings.Contains(p, "whatsapp") {
		return "📖 **Guía: Control de Accesos y Garita (`/visitas`)**\n\n" +
			"1. Entra a **Operaciones ➔ Control de Accesos** (`/visitas`) o desde la **App Móvil** en la sección Garita.\n" +
			"2. Registra el nombre del visitante, placa (opcional) y casa destino.\n" +
			"3. Usa el botón de **Notificar por WhatsApp (`wa.me`)** o Push Web para avisar al residente, y marca la salida (`FINALIZADA`) cuando el vehículo abandone el predio.", true
	}

	if strings.Contains(p, "flota") || strings.Contains(p, "360") || strings.Contains(p, "taller") || strings.Contains(p, "maquinaria") {
		return "📖 **Guía: Control de Flota Pesada y Maquinaria 360 (`/flota`)**\n\n" +
			"1. **Catálogo (`/flota/vehiculos`):** Registra primero los vehículos y maquinaria.\n" +
			"2. **Solicitudes y FSM (`/flota`):** Crea la solicitud de movimiento. La unidad pasa por el ciclo de 10 estados (Asignada ➔ Taller ➔ Garita Salida ➔ En Tránsito ➔ Llegada ➔ Cierre).\n" +
			"3. **Checklist 360 PWA (`/flota/inspeccion`):** El mecánico o guardia realiza la inspección táctil con captura fotográfica y lectura de horómetro/km.", true
	}

	if strings.Contains(p, "mapa") || strings.Contains(p, "gps") || strings.Contains(p, "ruta") || strings.Contains(p, "historial") {
		return "📖 **Guía: Mapa Operativo y Rastreo GPS (`/map`)**\n\n" +
			"1. Entra a **Centro de Mando ➔ Mapa Operativo** (`/map`).\n" +
			"2. **Pestaña En Vivo:** Muestra en tiempo real a los guardias que están transmitiendo su baliza desde la app móvil.\n" +
			"3. **Pestaña Historial:** Selecciona un guardia, fecha y rango de horas y pulsa **Buscar y Trazar Ruta**. El mapa segmenta automáticamente los tramos reales sin cruzar paredes ni edificios.", true
	}

	if strings.Contains(p, "residencial") || strings.Contains(p, "casa") || strings.Contains(p, "contrato") {
		return "📖 **Guía: Gestión Residencial y Casas (`/residencial`)**\n\n" +
			"1. Entra a **Operaciones ➔ Gestión Residencial** (`/residencial`).\n" +
			"2. Crea primero el **Residencial base** definiendo su Latitud, Longitud y Zoom por defecto (esto centra el mapa operativo).\n" +
			"3. Dentro del residencial seleccionado, registra las **Casas** e indica si cuentan con contrato de seguridad activo.", true
	}

	if strings.Contains(p, "app") || strings.Contains(p, "movil") || strings.Contains(p, "móvil") || strings.Contains(p, "apk") || strings.Contains(p, "celular") {
		return "📖 **Guía: App Móvil Android (`NCS365 Guarda`)**\n\n" +
			"• **Guardias:** Activa el switch **Patrullaje / GPS en Vivo** para transmitir en segundo plano (asegura en Ajustes de Android tener Ubicación en *Permitir todo el tiempo* y Batería en *Sin restricciones*). También tienes **Escanear QR**, **Visitas**, **Incidentes**, **Mis Turnos** y el botón rojo **SOS**.\n" +
			"• **Administradores / Supervisores:** Al iniciar sesión se abre la consola móvil de **Hermes Sentinel** (alertas en vivo, chat IA y GPS). En el menú superior puedes cambiar a **Modo Guarda** si necesitas cubrir un puesto.", true
	}

	return "📖 **Guía Rápida de Módulos NCS365 (`https://app.ncsecurity.net`):**\n\n" +
		"• **Centro de Mando (`/` y `/map`):** Resumen general, guardias GPS en vivo e historial de rutas.\n" +
		"• **Hermes Sentinel (`/hermes`):** Alertas autónomas, configuración de umbrales (`⚙️`) y Chat IA.\n" +
		"• **Operaciones:** Incidentes (`/incidentes`), Rondas (`/rondas`), Puntos QR (`/puntos-qr`), Garita (`/visitas`), Flota 360 (`/flota`) y Residenciales (`/residencial`).\n" +
		"• **Turnos y Personal:** Matriz GRID (`/programacion`) y Usuarios/Roles (`/usuarios`).\n\n" +
		"💡 *Pregúntame específicamente cómo hacer algo (ej. `¿cómo creo una ronda?`, `¿cómo programo turnos?` o `¿cómo uso la app móvil?`).*", true
}
