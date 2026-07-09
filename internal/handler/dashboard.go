package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/repository"
)

// ObtenerEstadisticasHandler devuelve el resumen general del dashboard
func ObtenerEstadisticasHandler(w http.ResponseWriter, r *http.Request) {
	// Extraer el residencial_id del contexto (Inyectado por AuthMiddleware)
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	stats, err := repository.ObtenerEstadisticasDashboard(r.Context(), residencialID)
	if err != nil {
		http.Error(w, "Error obteniendo estadísticas: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
