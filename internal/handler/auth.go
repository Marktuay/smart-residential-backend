package handler

import (
	"encoding/json"
	"net/http"
	"smart-residential-backend/internal/auth"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// LoginHandler procesa la autenticación y devuelve un JWT
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email y password son requeridos", http.StatusBadRequest)
		return
	}

	// Buscar usuario en la base de datos
	usuario, err := repository.ObtenerUsuarioPorEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Error del servidor", http.StatusInternalServerError)
		return
	}

	// Verificar si existe y si la contraseña coincide (bcrypt)
	if usuario == nil || bcrypt.CompareHashAndPassword([]byte(usuario.PasswordHash), []byte(req.Password)) != nil {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	if !usuario.Activo {
		http.Error(w, "Cuenta desactivada", http.StatusForbidden)
		return
	}

	// Generar JWT
	tokenString, err := auth.GenerarToken(usuario.ID, usuario.ResidencialID, usuario.Rol)
	if err != nil {
		http.Error(w, "Error al generar token", http.StatusInternalServerError)
		return
	}

	// Respuesta exitosa
	res := model.LoginResponse{
		Token:         tokenString,
		ResidencialID: usuario.ResidencialID,
		Rol:           usuario.Rol,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
