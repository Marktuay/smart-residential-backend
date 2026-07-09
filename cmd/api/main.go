package main

import (
	"log"
	"net/http"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/handler"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
)

func main() {
	if err := db.InicializarDB(); err != nil {
		log.Fatalf("Error inicializando la base de datos: %v", err)
	}
	defer db.DB.Close()

	mux := http.NewServeMux()

	// Rutas Públicas
	mux.Handle("POST /api/v1/login", http.HandlerFunc(handler.LoginHandler))

	// Rutas Privadas (Protegidas por JWT y RBAC)
	
	// Incidentes
	mux.Handle("GET /api/v1/incidentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.ListarIncidentesHandler))))
	mux.Handle("POST /api/v1/incidentes", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleResidente, model.RoleGuardia)(http.HandlerFunc(handler.CreateIncidenteHandler))))
	
	// Auditoría
	mux.Handle("GET /api/v1/audit-logs", middleware.AuthMiddleware(middleware.RBACMiddleware(model.RoleAdmin, model.RoleSisadmin)(http.HandlerFunc(handler.ListarLogsHandler))))

	// Envolver todo el enrutador con el middleware CORS
	handlerConCORS := middleware.CorsMiddleware(mux)

	log.Println("Servidor iniciado en el puerto :8080...")
	if err := http.ListenAndServe(":8080", handlerConCORS); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}
