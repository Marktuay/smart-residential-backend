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
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)

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

	if usuarioID == 0 {
		http.Error(w, "Usuario no identificado", http.StatusUnauthorized)
		return
	}

	// Si residencialID es GLOBAL o vacío, obtener el residencial del usuario
	if residencialID == "" || residencialID == "GLOBAL" {
		if headerRes := r.Header.Get("X-Residencial-ID"); headerRes != "" && headerRes != "GLOBAL" {
			residencialID = headerRes
		}
	}

	if err := repository.RegistrarUbicacion(r.Context(), residencialID, usuarioID, req.Latitud, req.Longitud); err != nil {
		http.Error(w, "Error guardando ubicación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// EliminarUbicacionHandler borra la posición activa del guardia al desactivar el rastreo o cerrar sesión
func EliminarUbicacionHandler(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := r.Context().Value(middleware.UsuarioIDKey).(int64)
	if !ok || usuarioID == 0 {
		http.Error(w, "Usuario no identificado", http.StatusUnauthorized)
		return
	}

	if err := repository.EliminarUbicacionActiva(r.Context(), usuarioID); err != nil {
		http.Error(w, "Error eliminando ubicación activa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}

// ObtenerUbicacionesHandler devuelve las últimas ubicaciones registradas
func ObtenerUbicacionesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)

	// Permitir consultar por query param si es SISADMIN o si se solicita filtro específico
	if qResID := r.URL.Query().Get("residencial_id"); qResID != "" {
		residencialID = qResID
	}

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
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)
	
	if qResID := r.URL.Query().Get("residencial_id"); qResID != "" {
		residencialID = qResID
	}
	
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
