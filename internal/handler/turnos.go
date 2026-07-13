package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
	"strconv"
)

// ListarTurnosHandler lista los turnos del residencial activo
func ListarTurnosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	turnos, err := repository.ObtenerTurnos(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener turnos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(turnos)
}

// CrearTurnoHandler crea un nuevo turno
func CrearTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var t model.Turno
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Error decodificando request: "+err.Error(), http.StatusBadRequest)
		return
	}

	t.ResidencialID = residencialID

	if err := repository.CrearTurno(r.Context(), &t); err != nil {
		http.Error(w, "Error creando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Actualizar auditoría con IP
	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_TURNO", "ADMIN", middleware.GetClientIP(r), "Se programó un turno para guardia ID: "+strconv.FormatInt(t.GuardiaID, 10))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// ActualizarTurnoHandler modifica un turno existente
func ActualizarTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de turno inválido", http.StatusBadRequest)
		return
	}

	var t model.Turno
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Error decodificando request: "+err.Error(), http.StatusBadRequest)
		return
	}

	t.ID = id
	t.ResidencialID = residencialID

	if err := repository.ActualizarTurno(r.Context(), &t); err != nil {
		http.Error(w, "Error actualizando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Actualizar auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZAR_TURNO", "ADMIN", middleware.GetClientIP(r), "Se actualizó el turno ID: "+idStr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// EliminarTurnoHandler cancela/borra un turno
func EliminarTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de turno inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarTurno(r.Context(), id, residencialID); err != nil {
		http.Error(w, "Error eliminando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Actualizar auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "ELIMINAR_TURNO", "ADMIN", middleware.GetClientIP(r), "Se canceló/eliminó el turno ID: "+idStr)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"mensaje":"Turno cancelado exitosamente"}`))
}
