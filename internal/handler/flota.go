package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarVehiculosHandler lista la flota de vehículos del residencial
func ListarVehiculosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	vehiculos, err := repository.ListarVehiculosFlota(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener flota de vehículos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if vehiculos == nil {
		vehiculos = []model.VehiculoFlota{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehiculos)
}

// CrearVehiculoHandler crea un nuevo vehículo o maquinaria en el catálogo
func CrearVehiculoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var v model.VehiculoFlota
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if v.CodigoUnidad == "" || v.TipoUnidad == "" {
		http.Error(w, "Código de unidad y tipo de unidad son requeridos", http.StatusBadRequest)
		return
	}

	v.ResidencialID = residencialID
	if v.Estado == "" {
		v.Estado = "OPERATIVO"
	}
	if v.QrCode == "" {
		v.QrCode = "QR-" + v.CodigoUnidad
	}

	if err := repository.CrearVehiculoFlota(r.Context(), &v); err != nil {
		http.Error(w, "Error al crear vehículo de flota: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_VEHICULO_FLOTA", "USER", middleware.GetClientIP(r), "Creó vehículo: "+v.CodigoUnidad)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}

// ListarSolicitudesHandler obtiene las solicitudes de la flota
func ListarSolicitudesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	solicitudes, err := repository.ListarSolicitudesFlota(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener solicitudes de flota: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if solicitudes == nil {
		solicitudes = []model.SolicitudFlota{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(solicitudes)
}

// CrearSolicitudHandler abre una nueva solicitud de traslado por Operaciones
func CrearSolicitudHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	usuarioIDVal := r.Context().Value(middleware.UsuarioIDKey)
	var usuarioID int64 = 1
	if idVal, ok := usuarioIDVal.(int64); ok && idVal > 0 {
		usuarioID = idVal
	}

	var s model.SolicitudFlota
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if s.PuntoOrigenID == "" || s.PuntoDestinoID == "" {
		http.Error(w, "Punto de origen y destino son requeridos", http.StatusBadRequest)
		return
	}

	s.ResidencialID = residencialID
	s.CreadoPor = usuarioID
	if s.CodigoSolicitud == "" {
		s.CodigoSolicitud = fmt.Sprintf("SOL-%d", time.Now().Unix())
	}

	if err := repository.CrearSolicitudFlota(r.Context(), &s); err != nil {
		http.Error(w, "Error al crear solicitud de flota: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_SOLICITUD_FLOTA", fmt.Sprintf("USER_%d", usuarioID), middleware.GetClientIP(r), "Aperturó solicitud: "+s.CodigoSolicitud)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

// AsignarSolicitudHandler permite a Transporte vincular vehículo y conductor
func AsignarSolicitudHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de solicitud inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		VehiculoID  int64 `json:"vehiculo_id"`
		ConductorID int64 `json:"conductor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if req.VehiculoID <= 0 || req.ConductorID <= 0 {
		http.Error(w, "Vehículo y Conductor válidos son requeridos", http.StatusBadRequest)
		return
	}

	if err := repository.AsignarSolicitudFlota(r.Context(), residencialID, id, req.VehiculoID, req.ConductorID); err != nil {
		http.Error(w, "Error al asignar solicitud: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ASIGNAR_SOLICITUD_FLOTA", "TRANSPORTE", middleware.GetClientIP(r), fmt.Sprintf("Asignó vehículo %d a solicitud %d", req.VehiculoID, id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// RegistrarInspeccion360Handler permite a Taller o Garita registrar una inspección 360
func RegistrarInspeccion360Handler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	usuarioIDVal := r.Context().Value(middleware.UsuarioIDKey)
	var usuarioID int64 = 1
	if idVal, ok := usuarioIDVal.(int64); ok && idVal > 0 {
		usuarioID = idVal
	}

	var insp model.Inspeccion360
	if err := json.NewDecoder(r.Body).Decode(&insp); err != nil {
		http.Error(w, "Solicitud inválida: "+err.Error(), http.StatusBadRequest)
		return
	}

	if insp.SolicitudID <= 0 || insp.Etapa == "" || insp.Resultado == "" {
		http.Error(w, "SolicitudID, Etapa y Resultado son requeridos", http.StatusBadRequest)
		return
	}

	insp.ResidencialID = residencialID
	insp.InspectorID = usuarioID

	if err := repository.RegistrarInspeccion360(r.Context(), &insp); err != nil {
		http.Error(w, "Error al registrar inspección 360: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Transición FSM automática según etapa y resultado
	var nuevoEstado model.EstadoSolicitudFlota
	switch insp.Etapa {
	case "TALLER_SALIDA":
		if insp.Resultado == "APROBADO" {
			nuevoEstado = model.EstadoSolicitudAprobadaTaller
		} else {
			nuevoEstado = model.EstadoSolicitudRechazadaTaller
		}
	case "GARITA_SALIDA":
		if insp.Resultado == "APROBADO" {
			nuevoEstado = model.EstadoSolicitudEnTransito
		} else {
			nuevoEstado = model.EstadoSolicitudRechazadaGaritaSalida
		}
	case "GARITA_LLEGADA":
		if insp.Resultado == "APROBADO" {
			nuevoEstado = model.EstadoSolicitudCheckinConforme
		} else {
			nuevoEstado = model.EstadoSolicitudCheckinConObservacion
		}
	case "TALLER_CIERRE":
		nuevoEstado = model.EstadoSolicitudCerradoFinalizado
	}

	if nuevoEstado != "" {
		_ = repository.ActualizarEstadoSolicitudFlotaFSM(r.Context(), residencialID, insp.SolicitudID, nuevoEstado)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "INSPECCION_360", fmt.Sprintf("USER_%d", usuarioID), middleware.GetClientIP(r), fmt.Sprintf("Registró 360 etapa %s resultado %s", insp.Etapa, insp.Resultado))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(insp)
}

// ObtenerInspeccionesHandler lista las inspecciones registradas para una solicitud
func ObtenerInspeccionesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de solicitud inválido", http.StatusBadRequest)
		return
	}

	inspecciones, err := repository.ObtenerInspeccionesPorSolicitud(r.Context(), residencialID, id)
	if err != nil {
		http.Error(w, "Error al obtener inspecciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if inspecciones == nil {
		inspecciones = []model.Inspeccion360{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inspecciones)
}

// TransicionEstadoHandler permite forzar transiciones o resolver incidentes
func TransicionEstadoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de solicitud inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		NuevoEstado model.EstadoSolicitudFlota `json:"nuevo_estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if err := repository.ActualizarEstadoSolicitudFlotaFSM(r.Context(), residencialID, id, req.NuevoEstado); err != nil {
		http.Error(w, "Error al actualizar estado: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "TRANSICION_FLOTA_FSM", "ADMIN", middleware.GetClientIP(r), fmt.Sprintf("Transicionó solicitud %d a %s", id, req.NuevoEstado))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
