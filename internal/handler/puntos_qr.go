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

// ListarPuntosQRHandler maneja la obtención de los puntos físicos
func ListarPuntosQRHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	puntos, err := repository.ListarPuntosQR(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener los puntos QR: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if puntos == nil {
		puntos = []model.PuntoQR{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(puntos)
}

// CrearPuntoQRHandler registra un nuevo punto físico o de casa
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
		if strings.Contains(err.Error(), "DUPLICATE_PUNTO_QR") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al crear el punto QR: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"CREAR_PUNTO_QR",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se creó el punto QR '%s' (Código: %s)", p.Nombre, p.CodigoQR),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// ActualizarPuntoQRHandler modifica un punto existente
func ActualizarPuntoQRHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de punto QR inválido", http.StatusBadRequest)
		return
	}

	var p model.PuntoQR
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Error decodificando payload", http.StatusBadRequest)
		return
	}

	p.ID = id
	p.ResidencialID = residencialID

	if err := repository.ActualizarPuntoQR(&p); err != nil {
		if strings.Contains(err.Error(), "DUPLICATE_PUNTO_QR") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al actualizar punto QR: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"ACTUALIZAR_PUNTO_QR",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se actualizó el punto QR #%d ('%s')", id, p.Nombre),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  "Punto QR actualizado correctamente",
		"punto_qr": p,
	})
}

// EliminarPuntoQRHandler elimina un punto QR
func EliminarPuntoQRHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de punto QR inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarPuntoQR(id, residencialID); err != nil {
		http.Error(w, "Error al eliminar punto QR: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(
		residencialID,
		"ELIMINAR_PUNTO_QR",
		actor,
		middleware.GetClientIP(r),
		fmt.Sprintf("Se eliminó el punto QR #%d", id),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Punto QR eliminado correctamente",
	})
}
