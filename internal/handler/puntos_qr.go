package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

// ListarPuntosQRHandler maneja la obtención de los puntos físicos
func ListarPuntosQRHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	puntos, err := repository.ListarPuntosQR(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener los puntos QR", http.StatusInternalServerError)
		return
	}

	if puntos == nil {
		puntos = []model.PuntoQR{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(puntos)
}

// CrearPuntoQRHandler permite a un Admin/Supervisor registrar un nuevo punto
func CrearPuntoQRHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var p model.PuntoQR
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Error decodificando payload", http.StatusBadRequest)
		return
	}

	p.ResidencialID = residencialID

	if err := repository.CrearPuntoQR(&p); err != nil {
		http.Error(w, "Error al crear el punto QR", http.StatusInternalServerError)
		return
	}

	// Registrar auditoría asíncrona
	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"CREA_PUNTO_QR",
		"usuario_mock", // Eventualmente se extraerá del token JWT
		middleware.GetClientIP(r),
		"Punto QR creado: "+p.Nombre,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}
