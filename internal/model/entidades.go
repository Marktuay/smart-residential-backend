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
