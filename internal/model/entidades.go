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
	ID            int64     `json:"id"`
	ResidencialID string    `json:"residencial_id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"` // No se expone en JSON
	Rol           Role      `json:"rol"`
	CreadoEn      time.Time `json:"creado_en"`
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
	ResidenteNombre        string  `json:"residente_nombre,omitempty"`
	TieneContratoSeguridad bool    `json:"tiene_contrato_seguridad"`
}

type ProgramaRonda struct {
	ID            int64   `json:"id"`
	ResidencialID string  `json:"residencial_id"`
	Nombre        string  `json:"nombre"`
	HoraInicio    string  `json:"hora_inicio"`
	HoraFin       string  `json:"hora_fin"`
	Recurrencia   string  `json:"recurrencia"`
	Puntos        []int64 `json:"puntos,omitempty"` // IDs de PuntosQR
}

type AsignacionRonda struct {
	ID            int64  `json:"id"`
	ResidencialID string `json:"residencial_id"`
	ProgramaID    int64  `json:"programa_id"`
	UsuarioID     int64  `json:"usuario_id"`
	Fecha         string `json:"fecha"`
	Estado        string `json:"estado"`
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

type DashboardStats struct {
	Activos         int `json:"activos"`
	Criticos        int `json:"criticos"`
	ResueltosHoy    int `json:"resueltos_hoy"`
	TotalHistorico  int `json:"total_historico"`
	RondasHoy       int `json:"rondas_hoy"`
	VisitasHoy      int `json:"visitas_hoy"`
	Complejos       int `json:"complejos"`
	CasasQR         int `json:"casas_qr"`
	Residentes      int `json:"residentes"`
	UsuariosActivos int `json:"usuarios_activos"`
}

type Casa struct {
	ID                     int64  `json:"id"`
	ResidencialID          string `json:"residencial_id"`
	NumeroCasa             string `json:"numero_casa"`
	Bloque                 string `json:"bloque"`
	Estado                 string `json:"estado"`
	TieneContratoSeguridad bool   `json:"tiene_contrato_seguridad"`
	CreatedAt              string `json:"created_at"`
}

type Residente struct {
	ID            int64     `json:"id"`
	ResidencialID string    `json:"residencial_id"`
	CasaID        *int64    `json:"casa_id"`
	NumeroCasa    string    `json:"numero_casa,omitempty"` // Campo extra para joins
	Nombre        string    `json:"nombre"`
	Telefono      string    `json:"telefono"`
	Email         string    `json:"email"`
	EsPropietario bool      `json:"es_propietario"`
	CreatedAt     time.Time `json:"created_at"`
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
	ID            int64  `json:"id"`
	ResidencialID string `json:"residencial_id"`
	GuardiaID     int64  `json:"guardia_id"`
	GuardiaEmail  string `json:"guardia_email,omitempty"`
	Fecha         string `json:"fecha"`          // YYYY-MM-DD
	HoraInicio    string `json:"hora_inicio"`    // HH:MM
	HoraFin       string `json:"hora_fin"`       // HH:MM
	Descripcion   string `json:"descripcion"`
	Ubicacion     string `json:"ubicacion"`
	Estado        string `json:"estado"`         // PROGRAMADO, EN_CURSO, FINALIZADO, CANCELADO
}

type SuscripcionPush struct {
	ID            int64  `json:"id"`
	ResidencialID string `json:"residencial_id"`
	UsuarioID     int64  `json:"usuario_id"`
	Endpoint      string `json:"endpoint"`
	P256dh        string `json:"p256dh"`
	Auth          string `json:"auth"`
}
