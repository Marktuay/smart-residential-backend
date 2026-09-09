package model

import "time"

// Roles de Flota y Checklist 360
const (
	RoleOperadorOperaciones Role = "OPERADOR_OPERACIONES"
	RoleGestorTransporte    Role = "GESTOR_TRANSPORTE"
	RoleMecanicoTaller      Role = "MECANICO_TALLER"
	RoleGuardiaGarita       Role = "GUARDIA_GARITA"
	RoleAuditorFlota        Role = "AUDITOR_FLOTA"
)

// Estados FSM de Solicitud de Flota
type EstadoSolicitudFlota string

const (
	EstadoSolicitudDraft                 EstadoSolicitudFlota = "DRAFT"
	EstadoSolicitudAsignada              EstadoSolicitudFlota = "ASIGNADA"
	EstadoSolicitudRechazadaTransporte   EstadoSolicitudFlota = "RECHAZADA_TRANSPORTES"
	EstadoSolicitudInspeccionTallerPend  EstadoSolicitudFlota = "INSPECCION_TALLER_PENDIENTE"
	EstadoSolicitudAprobadaTaller        EstadoSolicitudFlota = "APROBADA_TALLER"
	EstadoSolicitudRechazadaTaller       EstadoSolicitudFlota = "RECHAZADA_TALLER"
	EstadoSolicitudGaritaSalidaPendiente EstadoSolicitudFlota = "GARITA_SALIDA_PENDIENTE"
	EstadoSolicitudRechazadaGaritaSalida EstadoSolicitudFlota = "RECHAZADA_GARITA_SALIDA"
	EstadoSolicitudEnTransito            EstadoSolicitudFlota = "EN_TRANSITO"
	EstadoSolicitudGaritaLlegadaPend     EstadoSolicitudFlota = "GARITA_LLEGADA_PENDIENTE"
	EstadoSolicitudCheckinConforme       EstadoSolicitudFlota = "CHECKIN_CONFORME"
	EstadoSolicitudCheckinConObservacion EstadoSolicitudFlota = "CHECKIN_CON_OBSERVACION"
	EstadoSolicitudInvestigacion         EstadoSolicitudFlota = "INVESTIGACION_INCIDENTE"
	EstadoSolicitudInspeccionCierre      EstadoSolicitudFlota = "INSPECCION_CIERRE_TALLER"
	EstadoSolicitudCerradoFinalizado     EstadoSolicitudFlota = "CERRADO_FINALIZADO"
)

// VehiculoFlota representa una unidad de transporte o maquinaria amarilla
type VehiculoFlota struct {
	ID                int64     `json:"id"`
	ResidencialID     string    `json:"residencial_id"`
	CodigoUnidad      string    `json:"codigo_unidad"`
	TipoUnidad        string    `json:"tipo_unidad"` // CAMION, RETROEXCAVADORA, CABEZAL, MONTACARGAS, GRUA, OTRO
	MarcaModelo       string    `json:"marca_modelo"`
	Placa             string    `json:"placa"`
	QrCode            string    `json:"qr_code"`
	HorometroKmActual float64   `json:"horometro_km_actual"`
	Estado            string    `json:"estado"` // OPERATIVO, EN_MANTENIMIENTO, INACTIVO
	CreatedAt         time.Time `json:"created_at"`
}

// SolicitudFlota representa una solicitud de traslado y uso de vehículo
type SolicitudFlota struct {
	ID                  int64                `json:"id"`
	ResidencialID       string               `json:"residencial_id"`
	CodigoSolicitud     string               `json:"codigo_solicitud"`
	PuntoOrigenID       string               `json:"punto_origen_id"`
	PuntoDestinoID      string               `json:"punto_destino_id"`
	VehiculoID          *int64               `json:"vehiculo_id"`
	VehiculoCodigo      string               `json:"vehiculo_codigo,omitempty"`
	VehiculoTipo        string               `json:"vehiculo_tipo,omitempty"`
	ConductorID         *int64               `json:"conductor_id"`
	ConductorNombre     string               `json:"conductor_nombre,omitempty"`
	EstadoActual        EstadoSolicitudFlota `json:"estado_actual"`
	EquipoAdicionalJson string               `json:"equipo_adicional_json"`
	CreadoPor           int64                `json:"creado_por"`
	CreadoPorNombre     string               `json:"creado_por_nombre,omitempty"`
	CreadoEn            time.Time            `json:"creado_en"`
	ActualizadoEn       time.Time            `json:"actualizado_en"`
}

// Inspeccion360 representa una evaluación de estado en Taller o Garita
type Inspeccion360 struct {
	ID               int64                 `json:"id"`
	SolicitudID      int64                 `json:"solicitud_id"`
	ResidencialID    string                `json:"residencial_id"`
	Etapa            string                `json:"etapa"` // TALLER_SALIDA, GARITA_SALIDA, GARITA_LLEGADA, TALLER_CIERRE
	InspectorID      int64                 `json:"inspector_id"`
	InspectorNombre  string                `json:"inspector_nombre,omitempty"`
	Resultado        string                `json:"resultado"` // APROBADO, RECHAZADO, CON_OBSERVACION
	HorometroKm      float64               `json:"horometro_km"`
	NivelCombustible string                `json:"nivel_combustible"`
	Observaciones    string                `json:"observaciones"`
	FechaInspeccion  time.Time             `json:"fecha_inspeccion"`
	Detalles         []ItemInspeccion360 `json:"detalles,omitempty"`
}

// ItemInspeccion360 representa el detalle de cada punto evaluado en el 360
type ItemInspeccion360 struct {
	ID           int64  `json:"id"`
	InspeccionID int64  `json:"inspeccion_id"`
	Categoria    string `json:"categoria"` // MECANICA, LUCES, SEGURIDAD, EPP_CONDUCTOR, ESTRUCTURA_CARGA
	ItemNombre   string `json:"item_nombre"`
	EstadoItem   string `json:"estado_item"` // BUENO, REGULAR, MALO, NO_APLICA
	FotoUrl      string `json:"foto_url"`
}
