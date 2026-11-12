package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// ListarUsuariosHandler maneja la petición para listar usuarios con soporte de filtros y paginación
func ListarUsuariosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	search := r.URL.Query().Get("search")
	rol := r.URL.Query().Get("rol")
	activoStr := r.URL.Query().Get("activo")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	var activoPtr *bool
	if activoStr == "true" {
		val := true
		activoPtr = &val
	} else if activoStr == "false" {
		val := false
		activoPtr = &val
	}

	page := 0
	limit := 0
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	} else if page > 0 {
		limit = 10 // Límite por defecto si se especifica página
	}

	usuarios, total, err := repository.ListarUsuariosFiltro(residencialID, search, rol, activoPtr, page, limit)
	if err != nil {
		http.Error(w, "Error al obtener usuarios: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Si no se solicitó paginación explícita, devolver arreglo simple para compatibilidad hacia atrás
	if page == 0 {
		json.NewEncoder(w).Encode(usuarios)
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	resp := model.UsuarioPaginadoResponse{
		Items:      usuarios,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	json.NewEncoder(w).Encode(resp)
}

// CrearUsuarioHandler maneja el registro de nuevos usuarios
func CrearUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var req struct {
		Email          string     `json:"email"`
		NombreCompleto string     `json:"nombre_completo"`
		Telefono       string     `json:"telefono"`
		Cedula         string     `json:"cedula"`
		Password       string     `json:"password"`
		Rol            model.Role `json:"rol"`
		Activo         *bool      `json:"activo"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida: "+err.Error(), http.StatusBadRequest)
		return
	}

	callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
	if req.Rol == model.RoleSisadmin && callerRole != model.RoleSisadmin {
		http.Error(w, "Solo SISADMIN puede asignar el rol SISADMIN", http.StatusForbidden)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" || req.Rol == "" {
		http.Error(w, "Usuario/Email, contraseña y rol son requeridos", http.StatusBadRequest)
		return
	}

	activo := true
	if req.Activo != nil {
		activo = *req.Activo
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error al procesar la contraseña", http.StatusInternalServerError)
		return
	}

	nuevoUsuario := model.Usuario{
		ResidencialID:  residencialID,
		Email:          req.Email,
		NombreCompleto: strings.TrimSpace(req.NombreCompleto),
		Telefono:       strings.TrimSpace(req.Telefono),
		Cedula:         strings.TrimSpace(req.Cedula),
		PasswordHash:   string(hashedPassword),
		Rol:            req.Rol,
		Activo:         activo,
	}

	if err := repository.CrearUsuario(&nuevoUsuario); err != nil {
		http.Error(w, "Error al crear usuario (quizá el usuario ya existe): "+err.Error(), http.StatusConflict)
		return
	}

	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	detalles := fmt.Sprintf("Creó usuario '%s' (%s) con rol %s", nuevoUsuario.Email, nuevoUsuario.NombreCompleto, nuevoUsuario.Rol)
	go repository.RegistrarAuditoriaAsync(residencialID, "CREA_USUARIO", actor, middleware.GetClientIP(r), detalles)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(nuevoUsuario)
}

// ActualizarUsuarioHandler modifica datos y estado de un usuario existente
func ActualizarUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

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
		Email          string     `json:"email"`
		NombreCompleto string     `json:"nombre_completo"`
		Telefono       string     `json:"telefono"`
		Cedula         string     `json:"cedula"`
		Password       string     `json:"password"`
		Rol            model.Role `json:"rol"`
		Activo         *bool      `json:"activo"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida: "+err.Error(), http.StatusBadRequest)
		return
	}

	callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
	if req.Rol == model.RoleSisadmin && callerRole != model.RoleSisadmin {
		http.Error(w, "Solo SISADMIN puede asignar el rol SISADMIN", http.StatusForbidden)
		return
	}

	activo := true
	if req.Activo != nil {
		activo = *req.Activo
	}

	var passwordHash string
	if strings.TrimSpace(req.Password) != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error al procesar la contraseña", http.StatusInternalServerError)
			return
		}
		passwordHash = string(h)
	}

	if err := repository.ActualizarUsuario(id, residencialID, strings.TrimSpace(req.Email), strings.TrimSpace(req.NombreCompleto), strings.TrimSpace(req.Telefono), strings.TrimSpace(req.Cedula), string(req.Rol), activo, passwordHash); err != nil {
		http.Error(w, "Error actualizando usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	detalles := fmt.Sprintf("Actualizó datos del usuario ID %d (%s, rol %s, activo=%t)", id, req.Email, req.Rol, activo)
	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZA_USUARIO", actor, middleware.GetClientIP(r), detalles)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))
}

// CambiarEstadoUsuarioHandler activa o suspende a un usuario sin eliminarlo
func CambiarEstadoUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id == 0 {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var req struct {
		Activo bool `json:"activo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Solicitud inválida", http.StatusBadRequest)
		return
	}

	if err := repository.CambiarEstadoUsuario(id, residencialID, req.Activo); err != nil {
		http.Error(w, "Error al cambiar estado: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	accion := "ACTIVAR_USUARIO"
	if !req.Activo {
		accion = "DESACTIVAR_USUARIO"
	}
	detalles := fmt.Sprintf("Cambió estado de usuario ID %d a activo=%t", id, req.Activo)
	go repository.RegistrarAuditoriaAsync(residencialID, accion, actor, middleware.GetClientIP(r), detalles)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

// EliminarUsuarioHandler realiza un SoftDelete conservando el histórico de turnos y auditorías
func EliminarUsuarioHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

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

	if err := repository.EliminarUsuario(id, residencialID); err != nil {
		http.Error(w, "Error eliminando usuario: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	detalles := fmt.Sprintf("SoftDelete ejecutado para usuario ID %d", id)
	go repository.RegistrarAuditoriaAsync(residencialID, "ELIMINA_USUARIO", actor, middleware.GetClientIP(r), detalles)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}

// CargaMasivaUsuariosHandler procesa un archivo CSV o un array JSON de colaboradores
func CargaMasivaUsuariosHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, ok := r.Context().Value(middleware.ResidencialIDKey).(string)
	if !ok || residencialID == "" {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	var usuariosAListar []model.Usuario
	contentType := r.Header.Get("Content-Type")

	defaultPasswordHash, _ := bcrypt.GenerateFromPassword([]byte("NCS2026!"), bcrypt.DefaultCost)

	if strings.Contains(contentType, "multipart/form-data") {
		// Procesar archivo CSV
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			http.Error(w, "Error al procesar archivo: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Archivo 'file' es requerido", http.StatusBadRequest)
			return
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			http.Error(w, "Error leyendo archivo", http.StatusInternalServerError)
			return
		}

		reader := csv.NewReader(buf)
		reader.TrimLeadingSpace = true

		records, err := reader.ReadAll()
		if err != nil {
			http.Error(w, "Error al analizar CSV: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(records) < 2 {
			http.Error(w, "El archivo CSV está vacío o solo contiene encabezados", http.StatusBadRequest)
			return
		}

		// Encabezados esperados: nombre_completo, email, cedula, telefono, rol, password
		header := records[0]
		colIdx := map[string]int{}
		for i, col := range header {
			colIdx[strings.ToLower(strings.TrimSpace(col))] = i
		}

		for lineNum, row := range records[1:] {
			getVal := func(name string) string {
				if idx, found := colIdx[name]; found && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}

			email := getVal("email")
			if email == "" {
				email = getVal("usuario")
			}
			if email == "" {
				continue // Ignorar fila vacía
			}

			nombre := getVal("nombre_completo")
			if nombre == "" {
				nombre = getVal("nombre")
			}
			cedula := getVal("cedula")
			if cedula == "" {
				cedula = getVal("dni")
			}
			telefono := getVal("telefono")
			rolStr := strings.ToUpper(getVal("rol"))
			if rolStr == "" {
				rolStr = "GUARDIA"
			}
			callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
			if model.Role(rolStr) == model.RoleSisadmin && callerRole != model.RoleSisadmin {
				http.Error(w, "Solo SISADMIN puede asignar el rol SISADMIN en carga masiva", http.StatusForbidden)
				return
			}
			pwd := getVal("password")

			pwdHash := string(defaultPasswordHash)
			if pwd != "" {
				if h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost); err == nil {
					pwdHash = string(h)
				}
			}

			usuariosAListar = append(usuariosAListar, model.Usuario{
				ResidencialID:  residencialID,
				Email:          email,
				NombreCompleto: nombre,
				Cedula:         cedula,
				Telefono:       telefono,
				PasswordHash:   pwdHash,
				Rol:            model.Role(rolStr),
				Activo:         true,
			})
			_ = lineNum
		}
	} else {
		// Procesar JSON
		var reqItems []struct {
			Email          string `json:"email"`
			NombreCompleto string `json:"nombre_completo"`
			Cedula         string `json:"cedula"`
			Telefono       string `json:"telefono"`
			Rol            string `json:"rol"`
			Password       string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqItems); err != nil {
			http.Error(w, "JSON inválido: "+err.Error(), http.StatusBadRequest)
			return
		}

		for _, item := range reqItems {
			email := strings.TrimSpace(item.Email)
			if email == "" {
				continue
			}

			rolStr := strings.ToUpper(strings.TrimSpace(item.Rol))
			if rolStr == "" {
				rolStr = "GUARDIA"
			}
			callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
			if model.Role(rolStr) == model.RoleSisadmin && callerRole != model.RoleSisadmin {
				http.Error(w, "Solo SISADMIN puede asignar el rol SISADMIN en carga masiva", http.StatusForbidden)
				return
			}

			pwdHash := string(defaultPasswordHash)
			if strings.TrimSpace(item.Password) != "" {
				if h, err := bcrypt.GenerateFromPassword([]byte(item.Password), bcrypt.DefaultCost); err == nil {
					pwdHash = string(h)
				}
			}

			usuariosAListar = append(usuariosAListar, model.Usuario{
				ResidencialID:  residencialID,
				Email:          email,
				NombreCompleto: strings.TrimSpace(item.NombreCompleto),
				Cedula:         strings.TrimSpace(item.Cedula),
				Telefono:       strings.TrimSpace(item.Telefono),
				PasswordHash:   pwdHash,
				Rol:            model.Role(rolStr),
				Activo:         true,
			})
		}
	}

	if len(usuariosAListar) == 0 {
		http.Error(w, "No se encontraron colaboradores válidos para registrar", http.StatusBadRequest)
		return
	}

	procesados, err := repository.CrearUsuariosBatch(residencialID, usuariosAListar)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error procesando lote (procesados %d): %v", procesados, err), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uid, ok := r.Context().Value(middleware.UsuarioIDKey).(int64); ok && uid > 0 {
		actor = fmt.Sprintf("usuario_id:%d", uid)
	}
	detalles := fmt.Sprintf("Carga masiva completada: %d colaboradores procesados", procesados)
	go repository.RegistrarAuditoriaAsync(residencialID, "CARGA_MASIVA_USUARIOS", actor, middleware.GetClientIP(r), detalles)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"procesados":  procesados,
		"total_filas": len(usuariosAListar),
	})
}
