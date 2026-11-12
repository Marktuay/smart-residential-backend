package middleware

import (
	"context"
	"net/http"
	"smart-residential-backend/internal/auth"
	"smart-residential-backend/internal/db"
	"smart-residential-backend/internal/model"
	"strings"
)

type contextKey string

const (
	ResidencialIDKey contextKey = "residencialID"
	UserRoleKey      contextKey = "userRole"
	UsuarioIDKey     contextKey = "usuarioID"
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

		// Verificación de estado activo/no eliminado contra BD para revocación inmediata
		if db.DB != nil && claims.UsuarioID > 0 {
			var activo, eliminado bool
			errDB := db.DB.QueryRowContext(r.Context(), "SELECT COALESCE(activo, true), deleted_at IS NOT NULL FROM usuarios WHERE id = $1", claims.UsuarioID).Scan(&activo, &eliminado)
			if errDB != nil || !activo || eliminado {
				http.Error(w, "Sesión revocada o usuario inactivo", http.StatusUnauthorized)
				return
			}
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
		ctx = context.WithValue(ctx, UsuarioIDKey, claims.UsuarioID)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
