package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"smart-residential-backend/internal/middleware"
	"smart-residential-backend/internal/model"
	"smart-residential-backend/internal/repository"
)

func ObtenerResidencialHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)
	callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)

	if callerRole == model.RoleSisadmin {
		if qID := r.URL.Query().Get("residencial_id"); qID != "" {
			residencialID = qID
		}
	}

	if residencialID == "" || residencialID == "GLOBAL" {
		primerID, err := repository.ObtenerPrimerResidencialID(r.Context())
		if err == nil && primerID != "" {
			residencialID = primerID
		}
	}

	residencial, err := repository.ObtenerResidencial(residencialID)
	if err != nil {
		// Fallback al primer residencial registrado si el ID del contexto no existe
		primerID, errPrimer := repository.ObtenerPrimerResidencialID(r.Context())
		if errPrimer == nil && primerID != "" && primerID != residencialID {
			residencial, err = repository.ObtenerResidencial(primerID)
		}
		if err != nil {
			http.Error(w, "No se encontró configuración para el residencial solicitado", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residencial)
}

func ListarResidencialesHandler(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)

	if callerRole != model.RoleSisadmin {
		res, err := repository.ObtenerResidencial(residencialID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]model.Residencial{*res})
		return
	}

	residenciales, err := repository.ListarResidenciales(r.Context())
	if err != nil {
		http.Error(w, "Error al listar residenciales", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residenciales)
}

func CrearResidencialHandler(w http.ResponseWriter, r *http.Request) {
	var residencial model.Residencial
	if err := json.NewDecoder(r.Body).Decode(&residencial); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	if residencial.ID == "" || residencial.Nombre == "" {
		http.Error(w, "ID y Nombre son requeridos", http.StatusBadRequest)
		return
	}

	if err := repository.CrearResidencial(r.Context(), &residencial); err != nil {
		http.Error(w, "Error al crear residencial", http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync("SISADMIN", "CREAR_RESIDENCIAL", "SISADMIN", middleware.GetClientIP(r), "Se creó el residencial "+residencial.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(residencial)
}

// ListarCasasHandler lista casas con paginación y búsqueda o completas para dropdowns
func ListarCasasHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	q := r.URL.Query()
	isPaginado := q.Get("paginado") == "true" || q.Get("page") != "" || q.Get("limit") != "" || q.Get("search") != ""

	if isPaginado {
		page, _ := strconv.Atoi(q.Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit < 1 {
			limit = 10
		}
		search := q.Get("search")
		bloque := q.Get("bloque")
		estado := q.Get("estado")

		resp, err := repository.ListarCasasPaginadas(residencialID, search, bloque, estado, page, limit)
		if err != nil {
			http.Error(w, "Error al obtener casas paginadas: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Modo legado / catálogo completo
	casas, err := repository.ObtenerCasas(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener casas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(casas)
}

func RegistrarCasaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var casa model.Casa
	if err := json.NewDecoder(r.Body).Decode(&casa); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	casa.ResidencialID = residencialID

	if strings.TrimSpace(casa.NumeroCasa) == "" {
		http.Error(w, "El número de casa es obligatorio", http.StatusBadRequest)
		return
	}

	if casa.Estado == "" {
		casa.Estado = "OCUPADA"
	}

	id, err := repository.RegistrarCasa(casa)
	if err != nil {
		if strings.Contains(err.Error(), "DUPLICATE_CASA") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al registrar la casa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	casa.ID = id

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_CASA", actor, middleware.GetClientIP(r), fmt.Sprintf("Se registró la casa %s (Bloque: %s)", casa.NumeroCasa, casa.Bloque))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(casa)
}

func ActualizarCasaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de casa inválido", http.StatusBadRequest)
		return
	}

	var casa model.Casa
	if err := json.NewDecoder(r.Body).Decode(&casa); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	casa.ID = id
	casa.ResidencialID = residencialID

	if strings.TrimSpace(casa.NumeroCasa) == "" {
		http.Error(w, "El número de casa es obligatorio", http.StatusBadRequest)
		return
	}

	if err := repository.ActualizarCasa(casa); err != nil {
		if strings.Contains(err.Error(), "DUPLICATE_CASA") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al actualizar casa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZAR_CASA", actor, middleware.GetClientIP(r), fmt.Sprintf("Se actualizó la casa #%d (%s, Bloque: %s)", id, casa.NumeroCasa, casa.Bloque))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Casa actualizada correctamente",
		"casa":    casa,
	})
}

func EliminarCasaHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de casa inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarCasa(id, residencialID); err != nil {
		http.Error(w, "Error al eliminar casa: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ELIMINAR_CASA", actor, middleware.GetClientIP(r), fmt.Sprintf("Se eliminó (SoftDelete) la casa ID #%d", id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Casa eliminada correctamente",
	})
}

// ListarResidentesHandler lista residentes con paginación y búsqueda
func ListarResidentesHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	q := r.URL.Query()
	isPaginado := q.Get("paginado") == "true" || q.Get("page") != "" || q.Get("limit") != "" || q.Get("search") != "" || q.Get("es_propietario") != ""

	if isPaginado {
		page, _ := strconv.Atoi(q.Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit < 1 {
			limit = 10
		}
		search := q.Get("search")

		var esPropietario *bool
		if pStr := q.Get("es_propietario"); pStr != "" {
			val := (pStr == "true" || pStr == "1")
			esPropietario = &val
		}

		resp, err := repository.ListarResidentesPaginados(residencialID, search, esPropietario, page, limit)
		if err != nil {
			http.Error(w, "Error al obtener residentes paginados: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	residentes, err := repository.ObtenerResidentes(residencialID)
	if err != nil {
		http.Error(w, "Error al obtener residentes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(residentes)
}

func RegistrarResidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)

	var residente model.Residente
	if err := json.NewDecoder(r.Body).Decode(&residente); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	residente.ResidencialID = residencialID

	if strings.TrimSpace(residente.Nombre) == "" {
		http.Error(w, "El nombre del residente es obligatorio", http.StatusBadRequest)
		return
	}

	id, err := repository.RegistrarResidente(residente)
	if err != nil {
		if strings.Contains(err.Error(), "DUPLICATE_RESIDENTE") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al registrar el residente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	residente.ID = id

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "CREAR_RESIDENTE", actor, middleware.GetClientIP(r), fmt.Sprintf("Se registró el residente %s (Cédula: %s)", residente.Nombre, residente.Cedula))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(residente)
}

func ActualizarResidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de residente inválido", http.StatusBadRequest)
		return
	}

	var residente model.Residente
	if err := json.NewDecoder(r.Body).Decode(&residente); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	residente.ID = id
	residente.ResidencialID = residencialID

	if strings.TrimSpace(residente.Nombre) == "" {
		http.Error(w, "El nombre del residente es obligatorio", http.StatusBadRequest)
		return
	}

	if err := repository.ActualizarResidente(residente); err != nil {
		if strings.Contains(err.Error(), "DUPLICATE_RESIDENTE") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Error al actualizar residente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZAR_RESIDENTE", actor, middleware.GetClientIP(r), fmt.Sprintf("Se actualizó el residente #%d (%s, Cédula: %s)", id, residente.Nombre, residente.Cedula))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Residente actualizado correctamente",
		"residente": residente,
	})
}

func EliminarResidenteHandler(w http.ResponseWriter, r *http.Request) {
	residencialID := r.Context().Value(middleware.ResidencialIDKey).(string)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "ID de residente inválido", http.StatusBadRequest)
		return
	}

	if err := repository.EliminarResidente(id, residencialID); err != nil {
		http.Error(w, "Error al eliminar residente: "+err.Error(), http.StatusInternalServerError)
		return
	}

	actor := "ADMIN"
	if uID := r.Context().Value(middleware.UsuarioIDKey); uID != nil {
		actor = fmt.Sprintf("User_%v", uID)
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ELIMINAR_RESIDENTE", actor, middleware.GetClientIP(r), fmt.Sprintf("Se eliminó (SoftDelete) el residente ID #%d", id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Residente eliminado correctamente",
	})
}

func ActualizarConfiguracionHandler(w http.ResponseWriter, r *http.Request) {
	residencialID, _ := r.Context().Value(middleware.ResidencialIDKey).(string)

	var req struct {
		ID                    string  `json:"id"`
		LatitudCentro         float64 `json:"latitud_centro"`
		LongitudCentro        float64 `json:"longitud_centro"`
		ZoomDefecto           int     `json:"zoom_defecto"`
		ToleranciaRondaMins   int     `json:"tolerancia_ronda_mins"`
		FrecuenciaGpsSeg      int     `json:"frecuencia_gps_seg"`
		EmailAlertaIncidentes string  `json:"email_alerta_incidentes"`
		TelefonoEmergencia    string  `json:"telefono_emergencia"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Cuerpo de solicitud inválido", http.StatusBadRequest)
		return
	}

	callerRole, _ := r.Context().Value(middleware.UserRoleKey).(model.Role)
	if callerRole == model.RoleSisadmin {
		if qID := r.URL.Query().Get("residencial_id"); qID != "" {
			residencialID = qID
		} else if req.ID != "" && req.ID != "GLOBAL" {
			residencialID = req.ID
		}
	}

	if residencialID == "" || residencialID == "GLOBAL" {
		primerID, err := repository.ObtenerPrimerResidencialID(r.Context())
		if err == nil && primerID != "" {
			residencialID = primerID
		}
	}

	if req.ZoomDefecto < 1 || req.ZoomDefecto > 22 {
		req.ZoomDefecto = 13
	}
	if req.ToleranciaRondaMins < 1 {
		req.ToleranciaRondaMins = 15
	}
	if req.FrecuenciaGpsSeg < 5 {
		req.FrecuenciaGpsSeg = 10
	}

	err := repository.ActualizarConfiguracionResidencial(
		r.Context(),
		residencialID,
		req.LatitudCentro,
		req.LongitudCentro,
		req.ZoomDefecto,
		req.ToleranciaRondaMins,
		req.FrecuenciaGpsSeg,
		req.EmailAlertaIncidentes,
		req.TelefonoEmergencia,
	)
	if err != nil {
		http.Error(w, "Error al actualizar configuración: "+err.Error(), http.StatusInternalServerError)
		return
	}

	go repository.RegistrarAuditoriaAsync(residencialID, "ACTUALIZAR_CONFIGURACION", "user_mock", middleware.GetClientIP(r), "Se actualizó la configuración del residencial")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
