package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarProgramasRondasHandler lista los programas
func ListarProgramasRondasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	programas, err := repository.ObtenerProgramasRondas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener programas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if programas == nil {
		programas = []model.ProgramaRonda{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(programas)
}

// CrearProgramaRondaHandler crea un nuevo programa con checkpoints ordenados
func CrearProgramaRondaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var p model.ProgramaRonda
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Error decodificando request", http.StatusBadRequest)
		return
	}

	p.ResidencialID = residencialID

	if p.Nombre == "" {
		http.Error(w, "El nombre de la ruta o programa de ronda es obligatorio", http.StatusBadRequest)
		return
	}

	if err := repository.CrearProgramaRonda(r.Context(), &p); err != nil {
		http.Error(w, "Error creando el programa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"CREAR_PROGRAMA_RONDA",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se creó el programa de ronda '%s' con %d puntos de control", p.Nombre, len(p.Puntos)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// EliminarProgramaRondaHandler elimina un programa de patrullaje
func EliminarProgramaRondaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de programa inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarProgramaRonda(r.Context(), id, residencialID); err != nil {
		http.Error(w, "Error al eliminar programa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"ELIMINAR_PROGRAMA_RONDA",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se eliminó el programa de ronda ID #%d", id),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Programa de ronda eliminado correctamente",
	})
}

// ListarAsignacionesRondasHandler lista las asignaciones de rondas
func ListarAsignacionesRondasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	asignaciones, err := repository.ObtenerAsignacionesRondas(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener asignaciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if asignaciones == nil {
		asignaciones = []model.AsignacionRonda{}
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

	if a.ProgramaID <= 0 {
		http.Error(w, "Se debe seleccionar un programa de ronda", http.StatusBadRequest)
		return
	}
	if a.Fecha == "" {
		http.Error(w, "La fecha de la ronda es obligatoria", http.StatusBadRequest)
		return
	}

	if err := repository.CrearAsignacionRonda(r.Context(), &a); err != nil {
		http.Error(w, "Error creando la asignación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"ASIGNAR_RONDA",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se asignó la ronda #%d al guardia ID #%d para el día %s", a.ID, a.UsuarioID, a.Fecha),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

// RegistrarMarcacionRondaHandler procesa el escaneo en vivo de un código QR por parte de un guardia
func RegistrarMarcacionRondaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req struct {
		CodigoQR          string  `json:"codigo_qr"`
		AsignacionRondaID *int64  `json:"asignacion_ronda_id,omitempty"`
		Latitud           float64 `json:"latitud"`
		Longitud          float64 `json:"longitud"`
		Observacion       string  `json:"observacion"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.CodigoQR == "" {
		http.Error(w, "El código QR es obligatorio", http.StatusBadRequest)
		return
	}

	// Buscar el punto por su código QR
	punto, err := repository.ObtenerPuntoPorCodigoQR(residencialID, req.CodigoQR)
	if err != nil {
		http.Error(w, "Código QR no válido o no pertenece a este residencial", http.StatusNotFound)
		return
	}

	var guardiaID int64
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		guardiaID = uID.(int64)
	}

	marcacion := model.MarcacionRonda{
		ResidencialID:     residencialID,
		AsignacionRondaID: req.AsignacionRondaID,
		PuntoQRID:         punto.ID,
		GuardiaID:         guardiaID,
		Latitud:           req.Latitud,
		Longitud:          req.Longitud,
		Observacion:       req.Observacion,
	}

	if err := repository.RegistrarMarcacionRonda(r.Context(), &marcacion); err != nil {
		http.Error(w, "Error al registrar marcación: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := fmt.Sprintf("Guardia_%d", guardiaID)
	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"MARCACION_QR",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Marcación exitosa en punto '%s' (Casa: %s)", punto.Nombre, punto.NumeroCasa),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":                  true,
		"mensaje":                  "Marcación registrada exitosamente",
		"punto":                    punto,
		"tiene_contrato_seguridad": punto.TieneContratoSeguridad,
		"fecha_hora":               marcacion.FechaHora,
	})
}

// ListarMarcacionesRondasHandler lista los escaneos recientes
func ListarMarcacionesRondasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	limite := 25
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limite = l
		}
	}

	marcaciones, err := repository.ListarMarcacionesRondas(r.Context(), residencialID, limite)
	if err != nil {
		http.Error(w, "Error al obtener marcaciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marcaciones)
}
