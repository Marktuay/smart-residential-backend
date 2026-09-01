package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// ListarUsuariosHandler maneja la petición para listar todos los usuarios
func ListarUsuariosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	usuarios, err := repository.ListarUsuarios(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener usuarios", http.StatusInternalServerError)
		return
	}

	if usuarios == nil {
		usuarios = []model.Usuario{} // Evitar null en JSON
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usuarios)
}

// CrearUsuarioHandler maneja el registro de nuevos usuarios (administradores o guardias)
func CrearUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email    string     `json:"email"`
		Password string     `json:"password"`
		Rol      model.Role `json:"rol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Rol == "" {
		http.Error(w, "Email, password y rol son requeridos", http.StatusBadRequest)
		return
	}

	// Encriptar contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error al procesar la contraseña", http.StatusInternalServerError)
		return
	}

	nuevoUsuario := model.Usuario{
		ResidencialID: residencialID,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		Rol:           req.Rol,
	}

	err = repository.CrearUsuario(&nuevoUsuario)
	if err != nil {
		http.Error(w, "Error al crear usuario (quizá el email ya existe)", http.StatusInternalServerError)
		return
	}

	// Auditar acción
	go repository.RegistrarAuditoriaAsync(residencialID, "CREA_USUARIO", "user_mock", middleware.GetClientIP(r), "Creó usuario: "+req.Email+" con rol: "+string(req.Rol))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(nuevoUsuario)
}

// ActualizarUsuarioHandler modifica el correo, rol o clave de un usuario existente
func ActualizarUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "ID de usuario requerido", http.StatusBadRequest)
		return
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id == 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		Email    string     `json:"email"`
		Password string     `json:"password"`
		Rol      model.Role `json:"rol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	var passwordHash string
	if req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error al procesar la contraseña", http.StatusInternalServerError)
			return
		}
		passwordHash = string(h)
	}

	if err := repository.ActualizarUsuario(id, req.Email, string(req.Rol), passwordHash); err != nil {
		http.Error(w, "Error actualizando usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}

// EliminarUsuarioHandler elimina un usuario
func EliminarUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "ID de usuario requerido", http.StatusBadRequest)
		return
	}

	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id == 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarUsuario(id); err != nil {
		http.Error(w, "Error eliminando usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}
