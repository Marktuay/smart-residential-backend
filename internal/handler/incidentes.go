package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
	"strconv"
)

// CrearIncidenteHandler maneja la creación de incidentes
func CrearIncidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req model.Incidente
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.ResidencialID = residencialID
	// Si no especificaron quién lo reporta, usamos un valor por defecto
	if req.ReportadoPor == "" {
		req.ReportadoPor = "USUARIO_AUTENTICADO"
	}

	if err := repository.CrearIncidente(r.Context(), &req); err != nil {
		http.Error(w, "Error creando incidente", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// ListarIncidentesHandler maneja la consulta de incidentes
func ListarIncidentesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	incidentes, err := repository.ObtenerIncidentes(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error obteniendo incidentes", http.StatusInternalServerError)
		return
	}

	if incidentes == nil {
		incidentes = []model.Incidente{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidentes)
}

// ResolverIncidenteHandler actualiza el estado de un incidente a resuelto
func ResolverIncidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	
	// Obtenemos el ID del URL path. Ejemplo: /api/v1/incidentes/12/resolver
	// Asumimos uso nativo net/http mux patterns: /api/v1/incidentes/{id}/resolver
	idStr := r.PathValue("id")
	
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := repository.ResolverIncidente(r.Context(), residencialID, id); err != nil {
		http.Error(w, "Error resolviendo incidente", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
