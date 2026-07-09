package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/repository"
)

// ListarLogsHandler maneja la petición GET /api/v1/audit-logs
func ListarLogsHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	logs, err := repository.ListarLogs(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error consultando logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
