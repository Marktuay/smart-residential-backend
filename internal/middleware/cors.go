package middleware

import "net/http"

// CorsMiddleware agrega los headers necesarios para permitir peticiones desde aplicaciones frontend (SPA)
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Permitir todos los orígenes (en producción se debe limitar al dominio del frontend)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Residencial-ID")
		
		// Responder de forma exitosa a las peticiones OPTIONS pre-flight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
