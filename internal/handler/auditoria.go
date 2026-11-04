package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarLogsHandler maneja la petición GET /api/v1/audit-logs
func ListarLogsHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)
	rol, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)

	filtroResidencial := r.URL.Query().Get("residencial_id")
	if rol == model.RoleSisadmin {
		if filtroResidencial != "" {
			residencialID = filtroResidencial
		} else if residencialID == "" {
			residencialID = "GLOBAL"
		}
	}

	logs, err := repository.ListarLogs(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error consultando logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
