package model

import "time"

// HermesConfig define los parámetros de supervisión autónoma por residencial o global
type HermesConfig struct {
	ID                     int64     `json:"id"`
	ResidencialID          string    `json:"residencial_id"`
	Activo                 bool      `json:"activo"`
	IntervaloEvaluacionSeg int       `json:"intervalo_evaluacion_seg"`
	MaxInactividadGpsMins  int       `json:"max_inactividad_gps_mins"`
	AlertaSosActiva        bool      `json:"alerta_sos_activa"`
	AlertaRondasRetraso    bool      `json:"alerta_rondas_retraso"`
	AlertaFlotaDesvios     bool      `json:"alerta_flota_desvios"`
	WebhookNotificacion    string    `json:"webhook_notificacion"`
	GeminiAPIKey           string    `json:"gemini_api_key,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// HermesAlerta representa una anomalía o evento detectado por el controlador Hermes
type HermesAlerta struct {
	ID               int64      `json:"id"`
	ResidencialID    string     `json:"residencial_id"`
	TipoEvento       string     `json:"tipo_evento"` // INACTIVIDAD_GPS, BOTON_PANICO_SOS, RONDA_VENCIDA, INCIDENTE_CRITICO, FLOTA_ANOMALIA
	Severidad        string     `json:"severidad"`   // BAJA, MEDIA, ALTA, CRITICA
	Titulo           string     `json:"titulo"`
	Descripcion      string     `json:"descripcion"`
	MetadataJSON     string     `json:"metadata_json"`
	Estado           string     `json:"estado"` // PENDIENTE, NOTIFICADO, ATENDIDO, DESCARTADO
	AtendidoPor      string     `json:"atendido_por,omitempty"`
	FechaCreacion    time.Time  `json:"fecha_creacion"`
	FechaResolucion  *time.Time `json:"fecha_resolucion,omitempty"`
}

// HermesStatusResponse describe la salud y métricas del motor Hermes
type HermesStatusResponse struct {
	Activo                     bool      `json:"activo"`
	Version                    string    `json:"version"`
	UltimaEvaluacion           time.Time `json:"ultima_evaluacion"`
	TotalAlertasPendientes     int       `json:"total_alertas_pendientes"`
	TotalAlertasCriticas       int       `json:"total_alertas_criticas"`
	ResidencialesMonitoreados  int       `json:"residenciales_monitoreados"`
	UptimeSegundos             int64     `json:"uptime_segundos"`
	Mensaje                    string    `json:"mensaje"`
}

// ResolverAlertaRequest define el payload para atender una alerta
type ResolverAlertaRequest struct {
	Estado      string `json:"estado"`       // ATENDIDO, DESCARTADO
	Comentario  string `json:"comentario"`   // Observaciones de resolución
	AtendidoPor string `json:"atendido_por"` // Nombre o ID del supervisor
}

// HermesChatHistoryItem representa un turno previo de conversación para memoria multi-turno
type HermesChatHistoryItem struct {
	Role string `json:"role"` // "user" o "model" (también acepta "hermes")
	Text string `json:"text"`
}

// HermesChatRequest define el mensaje del supervisor hacia Hermes
type HermesChatRequest struct {
	Mensaje    string                  `json:"mensaje"`
	Historial  []HermesChatHistoryItem `json:"historial,omitempty"`
	RutaActual string                  `json:"ruta_actual,omitempty"`
}

// HermesChatResponse representa la respuesta analítica o generada por Gemini
type HermesChatResponse struct {
	Respuesta string                 `json:"respuesta"`
	Fuente    string                 `json:"fuente"` // "GEMINI_AI" o "HERMES_ANALYTICS"
	Fecha     string                 `json:"fecha"`
	Contexto  map[string]interface{} `json:"contexto,omitempty"`
}
