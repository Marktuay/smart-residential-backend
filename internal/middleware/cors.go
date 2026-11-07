package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CorsMiddleware agrega los headers necesarios para permitir peticiones desde aplicaciones frontend (SPA)
// soportando orígenes específicos de producción y cabeceras de correlación de Cloudflare.
func CorsMiddleware(next http.Handler) http.Handler {
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	defaultOrigins := []string{
		"https://app.ncsecurity.net",
		"https://ncsecurity.net",
		"http://localhost:3000",
		"http://localhost:3001",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigin := "*"

		if origin != "" {
			if allowedOriginsEnv != "" {
				for _, o := range strings.Split(allowedOriginsEnv, ",") {
					if strings.TrimSpace(o) == origin {
						allowedOrigin = origin
						break
					}
				}
			} else {
				for _, o := range defaultOrigins {
					if o == origin {
						allowedOrigin = origin
						break
					}
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Residencial-ID, CF-Connecting-IP, CF-Ray, Accept")
		w.Header().Set("Access-Control-Expose-Headers", "X-Cloudflare-Ray, CF-Ray, X-Residencial-ID")
		w.Header().Set("Access-Control-Max-Age", "86400") // Cache preflight 24 horas

		// Responder de forma exitosa a las peticiones OPTIONS pre-flight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
