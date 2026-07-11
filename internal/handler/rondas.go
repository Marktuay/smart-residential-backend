package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarProgramasRondasHandler lista los programas
func ListarProgramasRondasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	programas, err := repository.ObtenerProgramasRondas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener programas", http.StatusInternalServerError)
		return
	}

	if programas == nil {
		programas = []model.ProgramaRonda{} // Devolver array vacío en vez de null
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(programas)
}

// CrearProgramaRondaHandler crea un nuevo programa
func CrearProgramaRondaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var p model.ProgramaRonda
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Error decodificando request", http.StatusBadRequest)
		return
	}

	p.ResidencialID = residencialID

	if err := repository.CrearProgramaRonda(r.Context(), &p); err != nil {
		http.Error(w, "Error creando el programa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Auditar acción (Asíncrono)
	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_PROGRAMA_RONDA", "Usuario ID (JWT)", middleware.GetClientIP(r), "Se creó el programa: "+p.Nombre)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// ListarAsignacionesRondasHandler lista las asignaciones
func ListarAsignacionesRondasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	asignaciones, err := repository.ObtenerAsignacionesRondas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener asignaciones", http.StatusInternalServerError)
		return
	}

	if asignaciones == nil {
		asignaciones = []model.AsignacionRonda{} // Array vacío
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asignaciones)
}

// CrearAsignacionRondaHandler asigna un programa a un guardia
func CrearAsignacionRondaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var a model.AsignacionRonda
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Error decodificando request", http.StatusBadRequest)
		return
	}

	a.ResidencialID = residencialID

	if err := repository.CrearAsignacionRonda(r.Context(), &a); err != nil {
		http.Error(w, "Error creando la asignación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Auditar acción (Asíncrono)
	go repository.RegistrarAuditoriaAsync(residencialID, "ASIGNA_RONDA", "Usuario ID (JWT)", middleware.GetClientIP(r), "Se asignó la ronda ID: ")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}
