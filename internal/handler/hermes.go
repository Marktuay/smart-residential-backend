package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"smart-residential-backend/internal/hermes"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ObtenerHermesStatusHandler devuelve la salud y métricas operativas del controlador Hermes
func ObtenerHermesStatusHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	engine := hermes.GetEngine()

	status := engine.GetStatus(r.Context(), residencialID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ListarHermesAlertasHandler lista las anomalías detectadas por Hermes
func ListarHermesAlertasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	rol := r.Context().Value(middleware.UserRoleKey).(model.Role)

	// SISADMIN puede filtrar por cualquier residencial mediante query param
	if rol == model.RoleSisadmin {
		if qResID := r.URL.Query().Get("residencial_id"); qResID != "" {
			residencialID = qResID
		}
	}

	estado := r.URL.Query().Get("estado")
	severidad := r.URL.Query().Get("severidad")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	alertas, err := repository.ListarHermesAlertas(r.Context(), residencialID, estado, severidad, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error consultando alertas de Hermes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alertas)
}

// ResolverHermesAlertaHandler actualiza el estado de una alerta (ej: ATENDIDO o DESCARTADO)
func ResolverHermesAlertaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de alerta inválido", http.StatusBadRequest)
		return
	}

	var req model.ResolverAlertaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	nuevoEstado := strings.ToUpper(strings.TrimSpace(req.Estado))
	if nuevoEstado != "ATENDIDO" && nuevoEstado != "DESCARTADO" && nuevoEstado != "NOTIFICADO" {
		nuevoEstado = "ATENDIDO"
	}

	atendidoPor := req.AtendidoPor
	if atendidoPor == "" {
		atendidoPor = fmt.Sprintf("Usuario-%v", r.Context().Value(middleware.UsuarioIDKey))
	}

	if err := repository.ResolverHermesAlerta(r.Context(), id, residencialID, nuevoEstado, atendidoPor); err != nil {
		http.Error(w, fmt.Sprintf("Error resolviendo alerta: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mensaje":     "Alerta actualizada exitosamente",
		"id":          id,
		"nuevo_estado": nuevoEstado,
	})
}

// ObtenerHermesConfigHandler consulta la configuración de monitoreo
func ObtenerHermesConfigHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	cfg, err := repository.ObtenerHermesConfig(r.Context(), residencialID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error obteniendo configuración de Hermes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// ActualizarHermesConfigHandler guarda cambios en los parámetros del centinela
func ActualizarHermesConfigHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req model.HermesConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	req.ResidencialID = residencialID
	if req.IntervaloEvaluacionSeg < 15 {
		req.IntervaloEvaluacionSeg = 15 // Mínimo 15 segundos
	}
	if req.MaxInactividadGpsMins < 1 {
		req.MaxInactividadGpsMins = 3 // Mínimo 3 minutos
	}

	if err := repository.GuardarHermesConfig(r.Context(), &req); err != nil {
		http.Error(w, fmt.Sprintf("Error guardando configuración de Hermes: %v", err), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "HERMES_CONFIG_ACTUALIZADA",
		fmt.Sprintf("Usuario-%v", r.Context().Value(middleware.UsuarioIDKey)), "127.0.0.1",
		fmt.Sprintf("Intervalo: %ds, Tolerancia GPS: %dm", req.IntervaloEvaluacionSeg, req.MaxInactividadGpsMins))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// EjecutarCicloHermesHandler fuerza un escaneo inmediato de reglas operativas
func EjecutarCicloHermesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	engine := hermes.GetEngine()

	if err := engine.TriggerEvaluation(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("Error ejecutando ciclo: %v", err), http.StatusInternalServerError)
		return
	}

	status := engine.GetStatus(r.Context(), residencialID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"mensaje": "Ciclo de supervisión ejecutado exitosamente",
		"status":  status,
	})
}

// EnviarMensajeChatHermesHandler procesa consultas conversacionales con IA hacia Hermes
func EnviarMensajeChatHermesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req model.HermesChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	resp, err := hermes.ProcessChatMessage(r.Context(), residencialID, req.Mensaje)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error procesando consulta con Hermes: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
