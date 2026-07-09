package middleware

import (
	"net/http"
	"smart-residential-backend/internal/model"
)

// RBACMiddleware verifica que el usuario tenga uno de los roles permitidos
func RBACMiddleware(allowedRoles ...model.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ahora el rol se extrae de manera segura desde el contexto (inyectado por AuthMiddleware vía JWT)
			userRoleInter := r.Context().Value(UserRoleKey)
			if userRoleInter == nil {
				http.Error(w, "Rol no encontrado en el contexto (requiere AuthMiddleware previo)", http.StatusUnauthorized)
				return
			}
			
			userRole := userRoleInter.(model.Role)

			isAllowed := false
			for _, role := range allowedRoles {
				if userRole == role || userRole == model.RoleSisadmin { // SISADMIN tiene acceso a todo
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				http.Error(w, "Prohibido: Permisos insuficientes", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
