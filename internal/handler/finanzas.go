package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/repository"
)

// ObtenerResumenFinanzasProxy actúa como un proxy inverso ligero hacia la API de NestJS
func ObtenerResumenFinanzasProxy(w http.ResponseWriter, r *http.Request) {
	// 1. Extraer el ResidencialID del contexto
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "Residencial ID no encontrado en contexto", http.StatusUnauthorized)
		return
	}

	// 2. Obtener URL de la API de Finanzas desde variables de entorno o usar un fallback
	finanzasAPIURL := os.Getenv("EXTERNAL_FINANZAS_API_URL")
	if finanzasAPIURL == "" {
		finanzasAPIURL = "http://localhost:3001/api/v1/cobros" // Default port para NestJS
	}

	// 3. Crear petición HTTP hacia el servicio externo
	req, err := http.NewRequestWithContext(r.Context(), "GET", finanzasAPIURL+"/resumen", nil)
	if err != nil {
		http.Error(w, "Error creando petición interna", http.StatusInternalServerError)
		return
	}

	// 4. Inyectar el X-Residencial-ID para que la API externa sepa de qué tenant se trata
	req.Header.Set("X-Residencial-ID", residencialID)
	// Propagar el token de autorización si es necesario
	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	// 5. Ejecutar la petición
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Mock Data si la API externa no está levantada
		mockResponse(w, residencialID)
		return
	}
	defer resp.Body.Close()

	// 6. Leer y reenviar la respuesta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	// Auditoría
	go repository.RegistrarAuditoriaAsync(residencialID, "CONSULTA_FINANZAS", "Usuario ID (JWT)", middleware.GetClientIP(r), "Consulta de resumen financiero via proxy")
}

// mockResponse provee datos de prueba si el servicio externo no está disponible
func mockResponse(w http.ResponseWriter, residencialID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	mockData := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"residencial_id": residencialID,
			"total_recaudado": 15400.50,
			"total_morosidad": 3200.00,
			"tasa_cumplimiento": 85.5,
			"cobros_pendientes": 12,
			"fuente": "MOCK_DATA (API Externa inaccesible)",
		},
	}
	json.NewEncoder(w).Encode(mockData)
}
