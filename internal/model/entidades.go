package model

import "time"

type Role string

const (
	RoleSisadmin  Role = "SISADMIN"
	RoleAdmin     Role = "ADMIN"
	RoleResidente Role = "RESIDENTE"
	RoleGuardia   Role = "GUARDIA"
)

type Residencial struct {
	ID        string    `json:"id"`
	Nombre    string    `json:"nombre"`
	Direccion string    `json:"direccion"`
	CreatedAt time.Time `json:"created_at"`
}

type Incidente struct {
	ID            int64     `json:"id"`
	ResidencialID string    `json:"residencial_id"`
	Titulo        string    `json:"titulo"`
	Descripcion   string    `json:"descripcion"`
	Prioridad     string    `json:"prioridad"`
	Estado        string    `json:"estado"`
	CategoriaID   int       `json:"categoria_id"`
	CreadoEn      time.Time `json:"creado_en"`
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
	ID            int64   `json:"id"`
	ResidencialID string  `json:"residencial_id"`
	Nombre        string  `json:"nombre"`
	CodigoQR      string  `json:"codigo_qr"`
	Latitud       float64 `json:"latitud"`
	Longitud      float64 `json:"longitud"`
}

type ProgramaRonda struct {
	ID            int64  `json:"id"`
	ResidencialID string `json:"residencial_id"`
	Nombre        string `json:"nombre"`
	HoraInicio    string `json:"hora_inicio"`
	HoraFin       string `json:"hora_fin"`
	Recurrencia   string `json:"recurrencia"`
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
	Estado          string     `json:"estado"`
	FechaEsperada   string     `json:"fecha_esperada"`
	FechaIngreso    *time.Time `json:"fecha_ingreso,omitempty"`
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
