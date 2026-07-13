package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"

	"github.com/SherClockHolmes/webpush-go"
)

var (
	// Claves VAPID por defecto (se pueden sobreescribir desde variables de entorno)
	// Si no están configuradas, se autogenerarán en la inicialización
	VAPIDPublicKey  string
	VAPIDPrivateKey string
)

func InitVAPIDKeys() {
	// Generar claves automáticamente si están vacías
	if VAPIDPublicKey == "" || VAPIDPrivateKey == "" {
		pubKey, privKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			log.Fatalf("Error generando VAPID keys: %v", err)
		}
		VAPIDPublicKey = pubKey
		VAPIDPrivateKey = privKey
		log.Println("Claves VAPID autogeneradas exitosamente para Web Push.")
	}
}

// ObtenerVAPIDPublicKeyHandler expone la clave pública VAPID para el frontend
func ObtenerVAPIDPublicKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"public_key": VAPIDPublicKey,
	})
}

// RegistrarSuscripcionHandler registra una suscripción push para el usuario autenticado
func RegistrarSuscripcionHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	usuarioIDVal := r.Context().Value(middleware.UsuarioIDKey)
	if usuarioIDVal == nil {
		http.Error(w, "ID de usuario no encontrado en el contexto", http.StatusUnauthorized)
		return
	}
	usuarioID := usuarioIDVal.(int64)

	var req struct {
		Endpoint string `json:"endpoint"`
		P256dh   string `json:"p256dh"`
		Auth     string `json:"auth"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.Endpoint == "" || req.P256dh == "" || req.Auth == "" {
		http.Error(w, "endpoint, p256dh y auth son requeridos", http.StatusBadRequest)
		return
	}

	sub := &model.SuscripcionPush{
		ResidencialID: residencialID,
		UsuarioID:     usuarioID,
		Endpoint:      req.Endpoint,
		P256dh:        req.P256dh,
		Auth:          req.Auth,
	}

	if err := repository.GuardarSuscripcion(r.Context(), sub); err != nil {
		http.Error(w, "Error al guardar la suscripción: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

type EnviarNotificacionRequest struct {
	GuardiaID int64  `json:"guardia_id,omitempty"` // 0 o vacío envía a todos
	Titulo    string `json:"titulo"`
	Cuerpo    string `json:"cuerpo"`
}

// EnviarNotificacionHandler permite a Admin/Sisadmin disparar notificaciones push
func EnviarNotificacionHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var req EnviarNotificacionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.Titulo == "" || req.Cuerpo == "" {
		http.Error(w, "titulo y cuerpo son requeridos", http.StatusBadRequest)
		return
	}

	var subs []model.SuscripcionPush
	var err error

	if req.GuardiaID > 0 {
		subs, err = repository.ObtenerSuscripcionesPorUsuario(r.Context(), req.GuardiaID)
	} else {
		subs, err = repository.ObtenerSuscripcionesPorResidencial(r.Context(), residencialID)
	}

	if err != nil {
		http.Error(w, "Error obteniendo suscripciones: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Payload de notificación que el Service Worker entenderá
	payload, _ := json.Marshal(map[string]string{
		"title": req.Titulo,
		"body":  req.Cuerpo,
	})

	successCount := 0
	failedCount := 0

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, s, &webpush.Options{
			Subscriber:      "mailto:soporte@newcenturyni.com",
			VAPIDPublicKey:  VAPIDPublicKey,
			VAPIDPrivateKey: VAPIDPrivateKey,
			TTL:             60,
		})

		if err != nil {
			log.Printf("Error enviando push a sub %d: %v", sub.ID, err)
			failedCount++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			// La suscripción expiró o ya no existe, limpiarla
			repository.EliminarSuscripcionPorEndpoint(r.Context(), sub.Endpoint)
			failedCount++
		} else {
			successCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enviadas": successCount,
		"fallidas": failedCount,
	})
}
