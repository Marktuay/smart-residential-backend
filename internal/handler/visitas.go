package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarVisitasHandler retorna las visitas del residencial
func ListarVisitasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	visitas, err := repository.ObtenerVisitas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener visitas", http.StatusInternalServerError)
		return
	}

	if visitas == nil {
		visitas = []model.Visita{} // Evitar null
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(visitas)
}

// RegistrarVisitaHandler crea una nueva visita
func RegistrarVisitaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var v model.Visita
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "Error decodificando request", http.StatusBadRequest)
		return
	}

	v.ResidencialID = residencialID

	if err := repository.RegistrarVisita(r.Context(), &v); err != nil {
		http.Error(w, "Error registrando visita: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Auditar acción (Asíncrono)
	go repository.RegistrarAuditoriaAsync(residencialID, "REGISTRAR_VISITA", "Usuario ID (JWT)", middleware.GetClientIP(r), "Se pre-registró la visita de: "+v.NombreVisitante)

	// El estado ya lo establece el repository a "PENDIENTE"
	v.Estado = "PENDIENTE"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}
