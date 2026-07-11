package main

import (
	"log"
	"net/http"
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
	if err := repository.InitIncidentesDB(); err != nil {
		log.Printf("Advertencia: No se pudo inicializar la tabla de incidentes: %v", err)
	}

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

	// Geolocalización
	mux.Handle("POST /api/v1/tracking/location", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleGuardia)(http.HandlerFunc(handler.RegistrarUbicacionHandler))))
	mux.Handle("GET /api/v1/tracking/live", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin)(http.HandlerFunc(handler.ObtenerUbicacionesHandler))))

	// Residencial
	mux.Handle("GET /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarCasasHandler))))
	mux.Handle("POST /api/v1/casas", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarCasaHandler))))
	mux.Handle("GET /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarResidentesHandler))))
	mux.Handle("POST /api/v1/residentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.RegistrarResidenteHandler))))
	// Usuarios
	mux.Handle("GET /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarUsuariosHandler))))
	mux.Handle("POST /api/v1/usuarios", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.CrearUsuarioHandler))))

	// Envolver todo el enrutador con el middleware CORS
	handlerConCORS := middleware.CorsMiddleware(mux)

	log.Println("Servidor iniciado en el puerto :8080...")
	if err := http.ListenAndServe(":8080", handlerConCORS); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}
