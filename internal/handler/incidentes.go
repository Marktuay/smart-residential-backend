package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
	"time"
)

// ListarIncidentesHandler maneja la petición GET /api/v1/incidentes
func ListarIncidentesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	incidentes, err := repository.ListarIncidentes(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al consultar incidentes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CONSULTA_INCIDENTES", "user_mock", middleware.GetClientIP(r), "Se listaron los incidentes del residencial")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(incidentes)
}

// CreateIncidenteHandler maneja la petición POST /api/v1/incidentes
func CreateIncidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	
	var inc model.Incidente
	if err := json.NewDecoder(r.Body).Decode(&inc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if inc.Titulo == "" || inc.Descripcion == "" || inc.Prioridad == "" {
		http.Error(w, "titulo, descripcion y prioridad son campos obligatorios", http.StatusBadRequest)
		return
	}

	inc.ResidencialID = residencialID
	inc.CreadoEn = time.Now()
	if inc.Estado == "" {
		inc.Estado = "PENDIENTE"
	}
	
	if err := repository.CrearIncidente(r.Context(), &inc); err != nil {
		http.Error(w, "Error al crear incidente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CREA_INCIDENTE", "user_mock", middleware.GetClientIP(r), fmt.Sprintf("Creó un incidente con ID %d", inc.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inc)
}
