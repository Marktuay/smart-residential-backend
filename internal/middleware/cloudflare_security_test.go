package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"smart-residential-backend/internal/middleware"
	"testing"
)

func TestCloudflareSecurityHeaders(t *testing.T) {
	// Handler simulado
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	stack := middleware.CloudflareSecurityMiddleware(dummyHandler)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	stack.ServeHTTP(rr, req)

	// Verificar cabeceras de seguridad requeridas
	expectedHeaders := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "SAMEORIGIN",
		"X-XSS-Protection":          "1; mode=block",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Permissions-Policy":        "geolocation=(self), camera=(self), microphone=()",
	}

	for header, expectedValue := range expectedHeaders {
		got := rr.Header().Get(header)
		if got != expectedValue {
			t.Errorf("Header [%s] incorrecto. Esperado: '%s', obtenido: '%s'", header, expectedValue, got)
		}
	}
}

func TestCloudflareRayAndCountryPropagation(t *testing.T) {
	var capturedRay string
	var capturedCountry string
	var capturedIP string

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRay = middleware.GetCFRay(r)
		capturedCountry = middleware.GetCFCountry(r)
		capturedIP = middleware.GetClientIP(r)
		w.WriteHeader(http.StatusOK)
	})

	stack := middleware.CloudflareSecurityMiddleware(dummyHandler)

	req := httptest.NewRequest("GET", "/api/v1/incidentes", nil)
	req.Header.Set("CF-Ray", "8f12a34b5c6d7e8f-MIA")
	req.Header.Set("CF-IPCountry", "NI")
	req.Header.Set("CF-Connecting-IP", "165.98.120.45")

	rr := httptest.NewRecorder()
	stack.ServeHTTP(rr, req)

	if capturedRay != "8f12a34b5c6d7e8f-MIA" {
		t.Errorf("CF-Ray no capturado. Obtenido: %s", capturedRay)
	}

	if capturedCountry != "NI" {
		t.Errorf("CF-IPCountry no capturado. Obtenido: %s", capturedCountry)
	}

	if capturedIP != "165.98.120.45" {
		t.Errorf("ClientIP no capturado correctamente. Obtenido: %s", capturedIP)
	}

	// Verificar que CF-Ray se expuso en la respuesta
	if rr.Header().Get("X-Cloudflare-Ray") != "8f12a34b5c6d7e8f-MIA" {
		t.Errorf("Header X-Cloudflare-Ray no presente en respuesta: %s", rr.Header().Get("X-Cloudflare-Ray"))
	}
}

func TestGetClientIP_PriorityResolution(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "Prioridad 1: CF-Connecting-IP directo",
			headers: map[string]string{
				"CF-Connecting-IP": "190.212.80.12",
				"True-Client-IP":   "10.0.0.1",
				"X-Forwarded-For":  "192.168.1.1, 10.0.0.2",
			},
			remoteAddr: "172.68.1.1:44321",
			expectedIP: "190.212.80.12",
		},
		{
			name: "Prioridad 2: True-Client-IP de Cloudflare Enterprise si no hay CF-Connecting-IP",
			headers: map[string]string{
				"True-Client-IP":  "181.115.30.5",
				"X-Forwarded-For": "10.10.10.1",
			},
			remoteAddr: "172.68.1.1:44321",
			expectedIP: "181.115.30.5",
		},
		{
			name: "Prioridad 3: Primer salto de X-Forwarded-For con espacios y comas",
			headers: map[string]string{
				"X-Forwarded-For": " 200.85.160.25 , 172.70.10.1, 10.0.0.5",
			},
			remoteAddr: "127.0.0.1:8080",
			expectedIP: "200.85.160.25",
		},
		{
			name: "Prioridad 4: RemoteAddr nativo removiendo el puerto",
			headers: map[string]string{},
			remoteAddr: "192.168.1.50:58432",
			expectedIP: "192.168.1.50",
		},
		{
			name: "Prioridad 4b: IPv6 RemoteAddr nativo con corchetes y puerto",
			headers: map[string]string{},
			remoteAddr: "[2800:bf0:80::1]:61234",
			expectedIP: "2800:bf0:80::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := middleware.GetClientIP(req)
			if got != tt.expectedIP {
				t.Errorf("GetClientIP() = %v, esperado %v", got, tt.expectedIP)
			}
		})
	}
}

func TestCORS_CloudflareIntegration(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cors := middleware.CorsMiddleware(dummyHandler)

	// Petición OPTIONS Pre-flight desde frontend de producción
	req := httptest.NewRequest("OPTIONS", "/api/v1/visitas", nil)
	req.Header.Set("Origin", "https://app.ncsecurity.net")
	req.Header.Set("Access-Control-Request-Method", "POST")

	rr := httptest.NewRecorder()
	cors.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Preflight OPTIONS falló. Status obtenido: %d, esperado %d", rr.Code, http.StatusNoContent)
	}

	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.ncsecurity.net" {
		t.Errorf("CORS Origin esperado https://app.ncsecurity.net, obtenido: %s", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	exposeHeaders := rr.Header().Get("Access-Control-Expose-Headers")
	if exposeHeaders == "" || !containsSubstring(exposeHeaders, "X-Cloudflare-Ray") {
		t.Errorf("X-Cloudflare-Ray debe ser expuesto en CORS. Obtenido: %s", exposeHeaders)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
