package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
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

func ActualizarEstadoVisitaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de visita inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		Estado string `json:"estado"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	if req.Estado != "PENDIENTE" && req.Estado != "EN_CURSO" && req.Estado != "FINALIZADA" {
		http.Error(w, "Estado de visita inválido", http.StatusBadRequest)
		return
	}

	if err := repository.ActualizarEstadoVisita(r.Context(), residencialID, id, req.Estado); err != nil {
		http.Error(w, "Error actualizando estado de visita: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Registrar en auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZAR_VISITA", "user_mock", middleware.GetClientIP(r), "Se actualizó la visita ID: "+idStr+" al estado: "+req.Estado)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
