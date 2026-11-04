package model

import "time"

type Role string

const (
	RoleSisadmin          Role = "SISADMIN"
	RoleAdmin             Role = "ADMIN"
	RoleResidente         Role = "RESIDENTE"
	RoleGuardia           Role = "GUARDIA"
	RoleGuardiaPatrullero Role = "GUARDIA_PATRULLERO"
	RoleGuardiaMotorizado Role = "GUARDIA_MOTORIZADO"
	RoleOperadorC2        Role = "OPERADOR_C2"
	RoleSupervisor        Role = "SUPERVISOR"
)

type Residencial struct {
	ID                    string    `json:"id"`
	Nombre                string    `json:"nombre"`
	Direccion             string    `json:"direccion"`
	LatitudCentro         *float64  `json:"latitud_centro"`
	LongitudCentro        *float64  `json:"longitud_centro"`
	ZoomDefecto           int       `json:"zoom_defecto"`
	ToleranciaRondaMins   int       `json:"tolerancia_ronda_mins"`
	FrecuenciaGpsSeg      int       `json:"frecuencia_gps_seg"`
	EmailAlertaIncidentes string    `json:"email_alerta_incidentes"`
	TelefonoEmergencia    string    `json:"telefono_emergencia"`
	CreatedAt             time.Time `json:"created_at"`
}

type AuditLog struct {
	ID            int64     `json:"id"`
	ResidencialID string    `json:"residencial_id"`
	Accion        string    `json:"accion"`
	Actor         string    `json:"actor"`
	IP            string    `json:"ip"`
	Detalles      string    `json:"detalles"`
	CreatedAt     time.Time `json:"created_at"`
}

type Usuario struct {
	ID             int64      `json:"id"`
	ResidencialID  string     `json:"residencial_id"`
	Email          string     `json:"email"`
	NombreCompleto string     `json:"nombre_completo"`
	Telefono       string     `json:"telefono"`
	Cedula         string     `json:"cedula"`
	PasswordHash   string     `json:"-"` // No se expone en JSON
	Rol            Role       `json:"rol"`
	Activo         bool       `json:"activo"`
	CreadoEn       time.Time  `json:"creado_en"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type UsuarioPaginadoResponse struct {
	Items      []Usuario `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	TotalPages int       `json:"total_pages"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token         string `json:"token"`
	ResidencialID string `json:"residencial_id"`
	Rol           Role   `json:"rol"`
}

type PuntoQR struct {
	ID                     int64   `json:"id"`
	ResidencialID          string  `json:"residencial_id"`
	CasaID                 *int64  `json:"casa_id,omitempty"`
	Nombre                 string  `json:"nombre"`
	CodigoQR               string  `json:"codigo_qr"`
	Latitud                float64 `json:"latitud,omitempty"`
	Longitud               float64 `json:"longitud,omitempty"`
	NumeroCasa             string  `json:"numero_casa,omitempty"`
	Bloque                 string  `json:"bloque,omitempty"`
	ResidenteNombre        string  `json:"residente_nombre,omitempty"`
	TieneContratoSeguridad bool    `json:"tiene_contrato_seguridad"`
	UltimaMarcacion        string  `json:"ultima_marcacion,omitempty"`
	TotalMarcaciones       int     `json:"total_marcaciones,omitempty"`
	CreatedAt              string  `json:"created_at,omitempty"`
}

type ProgramaRonda struct {
	ID             int64     `json:"id"`
	ResidencialID  string    `json:"residencial_id"`
	Nombre         string    `json:"nombre"`
	HoraInicio     string    `json:"hora_inicio"`
	HoraFin        string    `json:"hora_fin"`
	Recurrencia    string    `json:"recurrencia"`
	ToleranciaMins int       `json:"tolerancia_mins"`
	Puntos         []int64   `json:"puntos,omitempty"` // IDs de PuntosQR
	PuntosDetalle  []PuntoQR `json:"puntos_detalle,omitempty"`
	TotalPuntos    int       `json:"total_puntos"`
	CreatedAt      string    `json:"created_at,omitempty"`
}

type AsignacionRonda struct {
	ID                     int64   `json:"id"`
	ResidencialID          string  `json:"residencial_id"`
	ProgramaID             int64   `json:"programa_id"`
	ProgramaNombre         string  `json:"programa_nombre,omitempty"`
	HoraInicio             string  `json:"hora_inicio,omitempty"`
	HoraFin                string  `json:"hora_fin,omitempty"`
	UsuarioID              int64   `json:"usuario_id"`
	GuardiaNombre          string  `json:"guardia_nombre,omitempty"`
	Fecha                  string  `json:"fecha"`
	Estado                 string  `json:"estado"`
	PuntosCompletados      int     `json:"puntos_completados"`
	PuntosTotal            int     `json:"puntos_total"`
	PorcentajeCumplimiento float64 `json:"porcentaje_cumplimiento"`
	CreatedAt              string  `json:"created_at,omitempty"`
}

type MarcacionRonda struct {
	ID                int64     `json:"id"`
	ResidencialID     string    `json:"residencial_id"`
	AsignacionRondaID *int64    `json:"asignacion_ronda_id,omitempty"`
	PuntoQRID         int64     `json:"punto_qr_id"`
	PuntoNombre       string    `json:"punto_nombre,omitempty"`
	NumeroCasa        string    `json:"numero_casa,omitempty"`
	GuardiaID         int64     `json:"guardia_id"`
	GuardiaNombre     string    `json:"guardia_nombre,omitempty"`
	FechaHora         time.Time `json:"fecha_hora"`
	Latitud           float64   `json:"latitud,omitempty"`
	Longitud          float64   `json:"longitud,omitempty"`
	Observacion       string    `json:"observacion,omitempty"`
}

type Visita struct {
	ID              int64      `json:"id"`
	ResidencialID   string     `json:"residencial_id"`
	NombreVisitante string     `json:"nombre_visitante"`
	CedulaVisitante string     `json:"cedula_visitante"`
	Estado          string     `json:"estado"`
	FechaEsperada   string     `json:"fecha_esperada"`
	NumeroCasa      string     `json:"numero_casa"`
	MedioIngreso    string     `json:"medio_ingreso"`
	PlacaVehiculo   string     `json:"placa_vehiculo"`
	FechaIngreso    *time.Time `json:"fecha_ingreso,omitempty"`
	FechaSalida     *time.Time `json:"fecha_salida,omitempty"`
}

type ActividadDia struct {
	Name    string `json:"name"`
	Accesos int    `json:"accesos"`
	Rondas  int    `json:"rondas"`
}

type NovedadReciente struct {
	ID     int64  `json:"id"`
	Tipo   string `json:"tipo"` // INCIDENTE, RONDA, VISITA
	Titulo string `json:"titulo"`
	Desc   string `json:"desc"`
	Nivel  string `json:"nivel,omitempty"`
	Fecha  string `json:"fecha"`
	Time   string `json:"time"`
}

type DashboardStats struct {
	Activos          int               `json:"activos"`
	Criticos         int               `json:"criticos"`
	ResueltosHoy     int               `json:"resueltos_hoy"`
	TotalHistorico   int               `json:"total_historico"`
	RondasHoy        int               `json:"rondas_hoy"`
	VisitasHoy       int               `json:"visitas_hoy"`
	Complejos        int               `json:"complejos"`
	CasasQR          int               `json:"casas_qr"`
	Residentes       int               `json:"residentes"`
	UsuariosActivos  int               `json:"usuarios_activos"`
	ActividadSemanal []ActividadDia    `json:"actividad_semanal"`
	UltimasNovedades []NovedadReciente `json:"ultimas_novedades"`
}

type Casa struct {
	ID                     int64      `json:"id"`
	ResidencialID          string     `json:"residencial_id"`
	NumeroCasa             string     `json:"numero_casa"`
	Bloque                 string     `json:"bloque"`
	Estado                 string     `json:"estado"`
	TieneContratoSeguridad bool       `json:"tiene_contrato_seguridad"`
	ResidentesNombres      string     `json:"residentes_nombres,omitempty"`
	ResidentesCount        int        `json:"residentes_count,omitempty"`
	Activo                 bool       `json:"activo"`
	CreatedAt              string     `json:"created_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}

type CasaPaginadaResponse struct {
	Items      []Casa `json:"items"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int    `json:"total_pages"`
}

type Residente struct {
	ID                 int64      `json:"id"`
	ResidencialID      string     `json:"residencial_id"`
	CasaID             *int64     `json:"casa_id,omitempty"`     // Para compatibilidad
	NumeroCasa         string     `json:"numero_casa,omitempty"` // Para compatibilidad
	Nombre             string     `json:"nombre"`
	Cedula             string     `json:"cedula"`
	Telefono           string     `json:"telefono"`
	TelefonoEmergencia string     `json:"telefono_emergencia"`
	Email              string     `json:"email"`
	EsPropietario      bool       `json:"es_propietario"`
	CasasAsignadasIDs  []int64    `json:"casas_asignadas_ids,omitempty"`
	CasasDetalle       []Casa     `json:"casas_detalle,omitempty"`
	Activo             bool       `json:"activo"`
	CreatedAt          time.Time  `json:"created_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type ResidentePaginadoResponse struct {
	Items      []Residente `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

type Incidente struct {
	ID              int64      `json:"id"`
	ResidencialID   string     `json:"residencial_id"`
	ReportadoPor    string     `json:"reportado_por"`
	Tipo            string     `json:"tipo"`
	Descripcion     string     `json:"descripcion"`
	NivelGravedad   string     `json:"nivel_gravedad"`
	Estado          string     `json:"estado"`
	EvidenciaURL    string     `json:"evidencia_url"`
	FechaCreacion   time.Time  `json:"fecha_creacion"`
	FechaResolucion *time.Time `json:"fecha_resolucion,omitempty"`
}

type Turno struct {
	ID             int64      `json:"id"`
	ResidencialID  string     `json:"residencial_id"`
	GuardiaID      int64      `json:"guardia_id"`
	GuardiaNombre  string     `json:"guardia_nombre,omitempty"`
	GuardiaEmail   string     `json:"guardia_email,omitempty"`
	GuardiaCedula  string     `json:"guardia_cedula,omitempty"`
	Fecha          string     `json:"fecha"`          // YYYY-MM-DD
	HoraInicio     string     `json:"hora_inicio"`    // HH:MM
	HoraFin        string     `json:"hora_fin"`       // HH:MM
	Horas          float64    `json:"horas"`          // Duración calculada en horas
	Descripcion    string     `json:"descripcion"`
	Ubicacion      string     `json:"ubicacion"`
	Estado         string     `json:"estado"`         // PROGRAMADO, EN_CURSO, COMPLETADO, AUSENTE, CANCELADO
	Asistencia     string     `json:"asistencia"`     // PENDIENTE, PRESENTE, TARDANZA, AUSENTE
	MarcadoEntrada *time.Time `json:"marcado_entrada,omitempty"`
	MarcadoSalida  *time.Time `json:"marcado_salida,omitempty"`
	CreadoPor      string     `json:"creado_por,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ProgramacionMasivaRequest struct {
	GuardiaIDs []int64  `json:"guardia_ids"`
	FechaDesde string   `json:"fecha_desde"` // YYYY-MM-DD
	FechaHasta string   `json:"fecha_hasta"` // YYYY-MM-DD
	HoraInicio string   `json:"hora_inicio"` // HH:MM
	HoraFin    string   `json:"hora_fin"`    // HH:MM
	TipoTurno  string   `json:"tipo_turno"`  // TD, TN, EXTRA
	Ubicacion  string   `json:"ubicacion"`
	Notas      string   `json:"notas"`
	DiasSemana []int    `json:"dias_semana"` // 0=Domingo, 1=Lunes, ... opcional
}

type ProgramacionMasivaResponse struct {
	Creados    int      `json:"creados"`
	Conflictos int      `json:"conflictos"`
	Mensajes   []string `json:"mensajes"`
}

type SuscripcionPush struct {
	ID            int64  `json:"id"`
	ResidencialID string `json:"residencial_id"`
	UsuarioID     int64  `json:"usuario_id"`
	Endpoint      string `json:"endpoint"`
	P256dh        string `json:"p256dh"`
	Auth          string `json:"auth"`
}
