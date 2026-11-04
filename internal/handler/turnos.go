package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// getActor obtiene una identificación legible del usuario en sesión
func getActor(r *http.Request) string {
	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	if role, ok := r.Context().Value(middleware.UserRoleKey).(model.Role); ok && role != "" {
		actor = fmt.Sprintf("%s (%s)", actor, role)
	}
	return actor
}

// ListarTurnosHandler lista los turnos del residencial activo
func ListarTurnosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	turnos, err := repository.ObtenerTurnos(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error al obtener turnos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(turnos)
}

// CrearTurnoHandler crea un nuevo turno validando anti-solapamiento horario
func CrearTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var t model.Turno
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Error decodificando request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if t.GuardiaID == 0 || t.Fecha == "" || t.HoraInicio == "" || t.HoraFin == "" {
		http.Error(w, "Guardia, fecha, hora_inicio y hora_fin son requeridos", http.StatusBadRequest)
		return
	}

	t.ResidencialID = residencialID
	actor := getActor(r)
	ip := middleware.GetClientIP(r)

	if err := repository.CrearTurno(r.Context(), &t, actor, ip); err != nil {
		if strings.Contains(err.Error(), "solapamiento detectado") || strings.Contains(err.Error(), "idénticas") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error creando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// ActualizarTurnoHandler modifica un turno existente
func ActualizarTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
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
	actor := getActor(r)
	ip := middleware.GetClientIP(r)

	if err := repository.ActualizarTurno(r.Context(), &t, actor, ip); err != nil {
		if strings.Contains(err.Error(), "solapamiento detectado") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error actualizando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// EliminarTurnoHandler cancela/borra un turno
func EliminarTurnoHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "ID de turno inválido", http.StatusBadRequest)
		return
	}

	actor := getActor(r)
	ip := middleware.GetClientIP(r)

	if err := repository.EliminarTurno(r.Context(), id, residencialID, actor, ip); err != nil {
		http.Error(w, "Error eliminando turno: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"cancelled"}`))
}

// ProgramacionMasivaHandler procesa la asignación masiva de turnos
func ProgramacionMasivaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var req model.ProgramacionMasivaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.GuardiaIDs) == 0 || req.FechaDesde == "" || req.FechaHasta == "" {
		http.Error(w, "Guardias seleccionados, fecha_desde y fecha_hasta son requeridos", http.StatusBadRequest)
		return
	}

	if req.HoraInicio == "" {
		req.HoraInicio = "07:00"
	}
	if req.HoraFin == "" {
		req.HoraFin = "19:00"
	}

	actor := getActor(r)
	ip := middleware.GetClientIP(r)

	resp, err := repository.ProgramarTurnosMasivos(r.Context(), residencialID, req, actor, ip)
	if err != nil {
		http.Error(w, "Error en programación masiva: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// MarcarAsistenciaHandler registra la asistencia de un turno
func MarcarAsistenciaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id == 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		Asistencia string `json:"asistencia"` // PRESENTE, TARDANZA, AUSENTE, FINALIZADO
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida: "+err.Error(), http.StatusBadRequest)
		return
	}

	actor := getActor(r)
	ip := middleware.GetClientIP(r)

	if err := repository.MarcarAsistenciaTurno(r.Context(), id, residencialID, req.Asistencia, actor, ip); err != nil {
		http.Error(w, "Error marcando asistencia: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

// ListarAuditoriaTurnosHandler devuelve el historial de auditoría específico de turnos
func ListarAuditoriaTurnosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	logs, err := repository.ListarAuditoriaTurnos(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error obteniendo auditoría de turnos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
