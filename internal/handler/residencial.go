package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

func ListarCasasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	casas, err := repository.ObtenerCasas(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener casas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(casas)
}

func RegistrarCasaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var casa model.Casa
	if err := json.NewDecoder(r.Body).Decode(&casa); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	casa.ResidencialID = residencialID

	if casa.Estado == "" {
		casa.Estado = "OCUPADA"
	}

	id, err := repository.RegistrarCasa(casa)
	if err != nil {
		http.Error(w, "Error al registrar la casa", http.StatusInternalServerError)
		return
	}

	casa.ID = id

	// Registrar en auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "REGISTRO_CASA", "user_mock", middleware.GetClientIP(r), "Se registró la casa " + casa.NumeroCasa)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(casa)
}

func ListarResidentesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	residentes, err := repository.ObtenerResidentes(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener residentes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residentes)
}

func RegistrarResidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var residente model.Residente
	if err := json.NewDecoder(r.Body).Decode(&residente); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	residente.ResidencialID = residencialID

	id, err := repository.RegistrarResidente(residente)
	if err != nil {
		http.Error(w, "Error al registrar el residente", http.StatusInternalServerError)
		return
	}

	residente.ID = id

	// Registrar en auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "REGISTRO_RESIDENTE", "user_mock", middleware.GetClientIP(r), "Se registró el residente " + residente.Nombre)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(residente)
}
