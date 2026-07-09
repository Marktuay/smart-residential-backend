package handler

import (
	"encoding/json"
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

	if err := repository.RegistrarUbicacion(r.Context(), residencialID, req.UsuarioID, req.Latitud, req.Longitud); err != nil {
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
