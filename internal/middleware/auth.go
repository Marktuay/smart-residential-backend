package middleware

import (
	"context"
	"net/http"
	"smart-residential-backend/internal/auth"
	"smart-residential-backend/internal/model"
	"strings"
)

type contextKey string

const (
	ResidencialIDKey contextKey = "residencialID"
	UserRoleKey      contextKey = "userRole"
)

// AuthMiddleware extrae y valida el JWT e inyecta los datos en el contexto
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Token de autorización requerido (Formato: Bearer <token>)", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidarToken(tokenString)
		if err != nil {
			http.Error(w, "Token inválido o expirado", http.StatusUnauthorized)
			return
		}

		// Inyectar datos en el contexto
		residencialID := claims.ResidencialID
		if claims.Rol == model.RoleSisadmin {
			if headerResID := r.Header.Get("X-Residencial-ID"); headerResID != "" {
				residencialID = headerResID
			}
		}

		ctx := context.WithValue(r.Context(), ResidencialIDKey, residencialID)
		ctx = context.WithValue(ctx, UserRoleKey, claims.Rol)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientIP es una función helper para extraer el header CF-Connecting-IP
func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
