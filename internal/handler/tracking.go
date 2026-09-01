package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/repository"
)

// RegistrarUbicacionRequest define la estructura de petición
type RegistrarUbicacionRequest struct {
	UsuarioID int64   `json:"usuario_id"`
	Latitud   float64 `json:"latitud"`
	Longitud  float64 `json:"longitud"`
}

// RegistrarUbicacionHandler recibe la ubicación bajo demanda del guardia
func RegistrarUbicacionHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req RegistrarUbicacionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	usuarioID := req.UsuarioID
	if usuarioID == 0 {
		if ctxUID, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok {
			usuarioID = ctxUID
		}
	}

	if err := repository.RegistrarUbicacion(r.Context(), residencialID, usuarioID, req.Latitud, req.Longitud); err != nil {
		http.Error(w, "Error guardando ubicación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// ObtenerUbicacionesHandler devuelve las últimas ubicaciones registradas
func ObtenerUbicacionesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	ubicaciones, err := repository.ObtenerUbicacionesActivas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error obteniendo ubicaciones", http.StatusInternalServerError)
		return
	}

	if ubicaciones == nil {
		ubicaciones = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ubicaciones)
}

// ObtenerHistorialTrackingHandler devuelve el historial de ubicaciones de un guardia en un rango de fechas
func ObtenerHistorialTrackingHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	
	// Extraer parámetros de la URL
	usuarioIDStr := r.URL.Query().Get("usuario_id")
	fechaInicio := r.URL.Query().Get("fecha_inicio")
	fechaFin := r.URL.Query().Get("fecha_fin")

	if usuarioIDStr == "" || fechaInicio == "" || fechaFin == "" {
		http.Error(w, "usuario_id, fecha_inicio y fecha_fin son requeridos", http.StatusBadRequest)
		return
	}

	var usuarioID int64
	// Parsear int
	fmt.Sscanf(usuarioIDStr, "%d", &usuarioID)

	historial, err := repository.ObtenerHistorialUsuario(r.Context(), residencialID, usuarioID, fechaInicio, fechaFin)
	if err != nil {
		http.Error(w, "Error obteniendo historial: " + err.Error(), http.StatusInternalServerError)
		return
	}

	if historial == nil {
		historial = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(historial)
}
