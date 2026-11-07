package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

const (
	CFRayKey     contextKey = "cfRay"
	CFCountryKey contextKey = "cfCountry"
	ClientIPKey  contextKey = "clientIP"
)

// CloudflareSecurityMiddleware aplica cabeceras de seguridad recomendadas por Cloudflare / OWASP
// y extrae metadatos de telemetría (CF-Ray, CF-IPCountry) al contexto HTTP.
func CloudflareSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Cabeceras de Seguridad Cloudflare / OWASP
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(self), camera=(self), microphone=()")

		// 2. Extracción de Metadatos Cloudflare
		cfRay := strings.TrimSpace(r.Header.Get("CF-Ray"))
		cfCountry := strings.TrimSpace(r.Header.Get("CF-IPCountry"))
		clientIP := GetClientIP(r)

		// Exponer CF-Ray en la respuesta para correlación de logs con Cloudflare WAF
		if cfRay != "" {
			w.Header().Set("X-Cloudflare-Ray", cfRay)
		}

		// 3. Inyectar en Contexto
		ctx := context.WithValue(r.Context(), CFRayKey, cfRay)
		ctx = context.WithValue(ctx, CFCountryKey, cfCountry)
		ctx = context.WithValue(ctx, ClientIPKey, clientIP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientIP extrae de forma robusta la dirección IP real del cliente detrás del proxy Cloudflare.
// Prioriza CF-Connecting-IP > True-Client-IP > primer salto de X-Forwarded-For > RemoteAddr (sin puerto).
func GetClientIP(r *http.Request) string {
	// 1. Cloudflare canonical connecting IP
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		return sanitizeIP(cfIP)
	}

	// 2. Cloudflare Enterprise True-Client-IP
	if trueIP := strings.TrimSpace(r.Header.Get("True-Client-IP")); trueIP != "" {
		return sanitizeIP(trueIP)
	}

	// 3. X-Forwarded-For (tomar el primer hop que corresponde al cliente original)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		hops := strings.Split(xff, ",")
		if len(hops) > 0 {
			firstHop := strings.TrimSpace(hops[0])
			if firstHop != "" {
				return sanitizeIP(firstHop)
			}
		}
	}

	// 4. RemoteAddr nativo (remover :puerto)
	remote := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remote); err == nil {
		return sanitizeIP(host)
	}

	return sanitizeIP(remote)
}

// sanitizeIP limpia caracteres de control o espacios para prevenir log-injection
func sanitizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	// Quitar corchetes IPv6 si están presentes [::1] -> ::1
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")
	return ip
}

// GetCFRay recupera el Ray ID de Cloudflare desde el contexto o el header
func GetCFRay(r *http.Request) string {
	if ray, ok := r.Context().Value(CFRayKey).(string); ok && ray != "" {
		return ray
	}
	return strings.TrimSpace(r.Header.Get("CF-Ray"))
}

// GetCFCountry recupera el país detectado por Cloudflare (ej. NI, US, CR)
func GetCFCountry(r *http.Request) string {
	if country, ok := r.Context().Value(CFCountryKey).(string); ok && country != "" {
		return country
	}
	return strings.TrimSpace(r.Header.Get("CF-IPCountry"))
}
