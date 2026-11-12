package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"smart-residential-backend/internal/auth"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/handler"
	"smart-residential-backend/internal/hermes"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

func main() {
	if err := auth.InitJWT(); err != nil {
		log.Fatalf("Error inicializando JWT: %v", err)
	}
	if err := db.InicializarDB(); err != nil {
		log.Fatalf("Error inicializando la base de datos: %v", err)
	}
	defer db.DB.Close()

	// Inicializar tablas necesarias (si no existen)
	if err := repository.InitUsuariosDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar columnas de usuarios: %v", err)
	}
	if err := repository.InitResidencialesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de residenciales: %v", err)
	}
	if err := repository.InitIncidentesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de incidentes: %v", err)
	}
	if err := repository.InitTurnosDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de turnos: %v", err)
	}
	if err := repository.InitPuntosQRDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de puntos QR: %v", err)
	}
	if err := repository.InitRondasDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar las tablas de rondas: %v", err)
	}
	if err := repository.InitPushDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de suscripciones push: %v", err)
	}
	if err := repository.InicializarTablasFlota(context.Background()); err != nil {
		log.Printf("Advertencia: No se pudo inicializar las tablas de flota: %v", err)
	}
	if err := repository.InitHermesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar las tablas de Hermes: %v", err)
	}
	handler.InitVAPIDKeys()

	// Iniciar Motor Autónomo Hermes Sentinel (supervisión en segundo plano)
	hermesEngine := hermes.GetEngine()
	hermesEngine.Start(context.Background(), 60*time.Second)
	defer hermesEngine.Stop()

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
	mux.Handle("PUT /api/v1/puntos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.ActualizarPuntoQRHandler))))
	mux.Handle("DELETE /api/v1/puntos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.EliminarPuntoQRHandler))))

	// Rondas
	mux.Handle("GET /api/v1/rondas/programas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarProgramasRondasHandler))))
	mux.Handle("POST /api/v1/rondas/programas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearProgramaRondaHandler))))
	mux.Handle("DELETE /api/v1/rondas/programas/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.EliminarProgramaRondaHandler))))
	mux.Handle("GET /api/v1/rondas/asignaciones", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarAsignacionesRondasHandler))))
	mux.Handle("POST /api/v1/rondas/asignaciones", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearAsignacionRondaHandler))))
	mux.Handle("POST /api/v1/rondas/marcacion", middleware.AuthMiddleware(http.HandlerFunc(handler.RegistrarMarcacionRondaHandler)))
	mux.Handle("GET /api/v1/rondas/marcaciones", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarMarcacionesRondasHandler))))

	// Visitas
	mux.Handle("GET /api/v1/visitas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleGuardia)(http.HandlerFunc(handler.ListarVisitasHandler))))
	mux.Handle("POST /api/v1/visitas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.RegistrarVisitaHandler))))
	mux.Handle("PUT /api/v1/visitas/{id}/estado", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleGuardia)(http.HandlerFunc(handler.ActualizarEstadoVisitaHandler))))

	// Geolocalización
	mux.Handle("POST /api/v1/tracking/location", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleGuardiaPatrullero, model.RoleGuardiaMotorizado, model.RoleOperadorC2, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarUbicacionHandler))))
	mux.Handle("DELETE /api/v1/tracking/location", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleGuardiaPatrullero, model.RoleGuardiaMotorizado, model.RoleOperadorC2, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarUbicacionHandler))))
	mux.Handle("GET /api/v1/tracking/live", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleGuardiaPatrullero, model.RoleGuardiaMotorizado, model.RoleOperadorC2, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerUbicacionesHandler))))
	mux.Handle("GET /api/v1/tracking/historial", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia, model.RoleGuardiaPatrullero, model.RoleGuardiaMotorizado, model.RoleOperadorC2, model.RoleSupervisor, model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerHistorialTrackingHandler))))

	// Notificaciones Push
	mux.Handle("GET /api/v1/push/vapid-public-key", http.HandlerFunc(handler.ObtenerVAPIDPublicKeyHandler))
	mux.Handle("POST /api/v1/push/subscribe", middleware.AuthMiddleware(http.HandlerFunc(handler.RegistrarSuscripcionHandler)))
	mux.Handle("POST /api/v1/push/send", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EnviarNotificacionHandler))))

	// Residencial
	mux.Handle("GET /api/v1/residenciales", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ListarResidencialesHandler))))
	mux.Handle("POST /api/v1/residenciales", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleSisadmin)(http.HandlerFunc(handler.CrearResidencialHandler))))
	mux.Handle("GET /api/v1/residencial/info", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ObtenerResidencialHandler))))
	mux.Handle("PUT /api/v1/residencial/config", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarConfiguracionHandler))))
	mux.Handle("GET /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarCasasHandler))))
	mux.Handle("POST /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarCasaHandler))))
	mux.Handle("PUT /api/v1/casas/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarCasaHandler))))
	mux.Handle("DELETE /api/v1/casas/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarCasaHandler))))
	mux.Handle("GET /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarResidentesHandler))))
	mux.Handle("POST /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarResidenteHandler))))
	mux.Handle("PUT /api/v1/residentes/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarResidenteHandler))))
	mux.Handle("DELETE /api/v1/residentes/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarResidenteHandler))))
	// Usuarios y Gestión de Personal
	mux.Handle("GET /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarUsuariosHandler))))
	mux.Handle("POST /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CrearUsuarioHandler))))
	mux.Handle("POST /api/v1/usuarios/bulk-csv", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CargaMasivaUsuariosHandler))))
	mux.Handle("PUT /api/v1/usuarios/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarUsuarioHandler))))
	mux.Handle("PATCH /api/v1/usuarios/{id}/estado", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CambiarEstadoUsuarioHandler))))
	mux.Handle("DELETE /api/v1/usuarios/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EliminarUsuarioHandler))))

	// Turnos y Programación
	mux.Handle("GET /api/v1/turnos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor, model.RoleGuardia)(http.HandlerFunc(handler.ListarTurnosHandler))))
	mux.Handle("POST /api/v1/turnos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.CrearTurnoHandler))))
	mux.Handle("POST /api/v1/turnos/masivo", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ProgramacionMasivaHandler))))
	mux.Handle("POST /api/v1/turnos/{id}/asistencia", middleware.AuthMiddleware(http.HandlerFunc(handler.MarcarAsistenciaHandler)))
	mux.Handle("GET /api/v1/turnos/auditoria", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ListarAuditoriaTurnosHandler))))
	mux.Handle("PUT /api/v1/turnos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ActualizarTurnoHandler))))
	mux.Handle("DELETE /api/v1/turnos/{id}", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.EliminarTurnoHandler))))

	// Módulo Flota y Checklist 360
	mux.Handle("GET /api/v1/flota/vehiculos", middleware.AuthMiddleware(http.HandlerFunc(handler.ListarVehiculosHandler)))
	mux.Handle("POST /api/v1/flota/vehiculos", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleGestorTransporte, model.RoleSupervisor)(http.HandlerFunc(handler.CrearVehiculoHandler))))
	mux.Handle("GET /api/v1/flota/solicitudes", middleware.AuthMiddleware(http.HandlerFunc(handler.ListarSolicitudesHandler)))
	mux.Handle("POST /api/v1/flota/solicitudes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleOperadorOperaciones, model.RoleSupervisor, model.RoleGestorTransporte)(http.HandlerFunc(handler.CrearSolicitudHandler))))
	mux.Handle("PUT /api/v1/flota/solicitudes/{id}/asignar", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleGestorTransporte, model.RoleSupervisor)(http.HandlerFunc(handler.AsignarSolicitudHandler))))
	mux.Handle("POST /api/v1/flota/solicitudes/{id}/inspeccion", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleMecanicoTaller, model.RoleGuardiaGarita, model.RoleGuardia, model.RoleSupervisor)(http.HandlerFunc(handler.RegistrarInspeccion360Handler))))
	mux.Handle("GET /api/v1/flota/solicitudes/{id}/inspecciones", middleware.AuthMiddleware(http.HandlerFunc(handler.ObtenerInspeccionesHandler)))
	mux.Handle("PUT /api/v1/flota/solicitudes/{id}/estado", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleGestorTransporte, model.RoleAuditorFlota, model.RoleSupervisor)(http.HandlerFunc(handler.TransicionEstadoHandler))))

	// Controlador Autónomo Hermes
	mux.Handle("GET /api/v1/hermes/status", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ObtenerHermesStatusHandler))))
	mux.Handle("GET /api/v1/hermes/alertas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ListarHermesAlertasHandler))))
	mux.Handle("PUT /api/v1/hermes/alertas/{id}/resolver", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.ResolverHermesAlertaHandler))))
	mux.Handle("GET /api/v1/hermes/config", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ObtenerHermesConfigHandler))))
	mux.Handle("PUT /api/v1/hermes/config", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ActualizarHermesConfigHandler))))
	mux.Handle("POST /api/v1/hermes/evaluar", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.EjecutarCicloHermesHandler))))
	mux.Handle("POST /api/v1/hermes/chat", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin, model.RoleSupervisor)(http.HandlerFunc(handler.EnviarMensajeChatHermesHandler))))

	// Envolver todo el enrutador con los middlewares de Cloudflare Security y CORS
	handlerConSeguridad := middleware.CloudflareSecurityMiddleware(middleware.CorsMiddleware(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado en el puerto :%s...\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), handlerConSeguridad); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}
