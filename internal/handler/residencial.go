package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

func ObtenerResidencialHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	residencial, err := repository.ObtenerResidencial(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener residencial", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residencial)
}

func ListarResidencialesHandler(w http.ResponseWriter, r *http.Request) {
	// Solo debería acceder a esto un SISADMIN, o en este endpoint se retornan todos.
	residenciales, err := repository.ListarResidenciales(r.Context())
	if err != nil {
		http.Error(w, "Error al listar residenciales", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residenciales)
}

func CrearResidencialHandler(w http.ResponseWriter, r *http.Request) {
	var residencial model.Residencial
	if err := json.NewDecoder(r.Body).Decode(&residencial); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	if residencial.ID == "" || residencial.Nombre == "" {
		http.Error(w, "ID y Nombre son requeridos", http.StatusBadRequest)
		return
	}

	if err := repository.CrearResidencial(r.Context(), &residencial); err != nil {
		http.Error(w, "Error al crear residencial", http.StatusInternalServerError)
		return
	}

	// Opcional: Registrar auditoría
	go repository.RegistrarAuditoriaAsync("SISADMIN", "SISADMIN", "CREAR_RESIDENCIAL", "NA", "Se creó el residencial "+residencial.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(residencial)
}

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
