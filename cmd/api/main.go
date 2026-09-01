package main

import (
	"context"
	"log"
	"net/http"
	"time"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/handler"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

func main() {
	if err := db.InicializarDB(); err != nil {
		log.Fatalf("Error inicializando la base de datos: %v", err)
	}
	defer db.DB.Close()

	// Inicializar tablas necesarias (si no existen)
	if err := repository.InitResidencialesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de residenciales: %v", err)
	}
	if err := repository.InitIncidentesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de incidentes: %v", err)
	}
	if err := repository.InitTurnosDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de turnos: %v", err)
	}
	if err := repository.InitPushDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de suscripciones push: %v", err)
	}
	handler.InitVAPIDKeys()

	// Iniciar Garbage Collector (cada 24h) para el historial de ubicaciones
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			ctx := context.Background()
			eliminados, err := repository.EliminarUbicacionesAntiguas(ctx, 60)
			if err != nil {
				log.Printf("Error en Garbage Collector de ubicaciones: %v", err)
			} else {
				log.Printf("Garbage Collector ejecutado: %d ubicaciones antiguas eliminadas", eliminados)
			}
			<-ticker.C
		}
	}()

	mux := http.NewServeMux()

	// Rutas Públicas
	mux.Handle("POST /api/v1/login", http.HandlerFunc(handler.LoginHandler))

	// Rutas Privadas (Protegidas por JWT y RBAC)
	
	// Incidentes
	mux.Handle("GET /api/v1/incidentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.ListarIncidentesHandler))))
	mux.Handle("POST /api/v1/incidentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.CrearIncidenteHandler))))
	mux.Handle("PUT /api/v1/incidentes/{id}/resolver", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.ResolverIncidenteHandler))))

	
	// Auditoría
	mux.Handle("GET /api/v1/audit-logs", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarLogsHandler))))

	// Dashboard
	mux.Handle("GET /api/v1/dashboard/stats", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerEstadisticasHandler))))

	// Finanzas (Proxy a NestJS)
	mux.Handle("GET /api/v1/finanzas/resumen", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerResumenFinanzasProxy))))

	// Puntos QR
	mux.Handle("GET /api/v1/puntos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarPuntosQRHandler))))
	mux.Handle("POST /api/v1/puntos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearPuntoQRHandler))))

	// Rondas
	mux.Handle("GET /api/v1/rondas/programas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarProgramasRondasHandler))))
	mux.Handle("POST /api/v1/rondas/programas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearProgramaRondaHandler))))
	mux.Handle("GET /api/v1/rondas/asignaciones", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarAsignacionesRondasHandler))))
	mux.Handle("POST /api/v1/rondas/asignaciones", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearAsignacionRondaHandler))))

	// Visitas
	mux.Handle("GET /api/v1/visitas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleGuardia)(http.HandlerFunc(handler.ListarVisitasHandler))))
	mux.Handle("POST /api/v1/visitas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.RegistrarVisitaHandler))))
	mux.Handle("PUT /api/v1/visitas/{id}/estado", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleGuardia)(http.HandlerFunc(handler.ActualizarEstadoVisitaHandler))))

	// Geolocalización
	mux.Handle("POST /api/v1/tracking/location", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarUbicacionHandler))))
	mux.Handle("DELETE /api/v1/tracking/location", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarUbicacionHandler))))
	mux.Handle("GET /api/v1/tracking/live", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerUbicacionesHandler))))
	mux.Handle("GET /api/v1/tracking/historial", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerHistorialTrackingHandler))))

	// Notificaciones Push
	mux.Handle("GET /api/v1/push/vapid-public-key", http.HandlerFunc(handler.ObtenerVAPIDPublicKeyHandler))
	mux.Handle("POST /api/v1/push/subscribe", middleware.AuthMiddleware(http.HandlerFunc(handler.RegistrarSuscripcionHandler)))
	mux.Handle("POST /api/v1/push/send", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EnviarNotificacionHandler))))

	// Residencial
	mux.Handle("GET /api/v1/residenciales", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleSisadmin)(http.HandlerFunc(handler.ListarResidencialesHandler))))
	mux.Handle("POST /api/v1/residenciales", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleSisadmin)(http.HandlerFunc(handler.CrearResidencialHandler))))
	mux.Handle("GET /api/v1/residencial/info", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ObtenerResidencialHandler))))
	mux.Handle("PUT /api/v1/residencial/config", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarConfiguracionHandler))))
	mux.Handle("GET /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarCasasHandler))))
	mux.Handle("POST /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarCasaHandler))))
	mux.Handle("GET /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarResidentesHandler))))
	mux.Handle("POST /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarResidenteHandler))))
	// Usuarios
	mux.Handle("GET /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarUsuariosHandler))))
	mux.Handle("POST /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CrearUsuarioHandler))))

	// Turnos
	mux.Handle("GET /api/v1/turnos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarTurnosHandler))))
	mux.Handle("POST /api/v1/turnos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CrearTurnoHandler))))
	mux.Handle("PUT /api/v1/turnos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarTurnoHandler))))
	mux.Handle("DELETE /api/v1/turnos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarTurnoHandler))))

	// Envolver todo el enrutador con el middleware CORS
	handlerConCORS := middleware.CorsMiddleware(mux)

	log.Println("Servidor iniciado en el puerto :8080...")
	if err := http.ListenAndServe(":8080", handlerConCORS); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}
