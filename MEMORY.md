# MEMORIA DEL PROYECTO SMART RESIDENTIAL (NCS360)

**Última actualización:** 28 de Julio, 2026  
**Cliente / Proyecto:** Smart Residential (NC Security / NCS360)  
**Dominio Principal:** `ncsecurity.net`  
**GCP Project ID:** `skilful-gantry-345115`  
**Cuenta GCP:** `informatica@blok-on.com`  
**GitHub:** `Marktuay`  

---

## 🚀 1. Resumen Ejecutivo del Estado del Proyecto

Se ha completado la migración y despliegue en producción de la solución integral **Smart Residential**:
* **Backend:** Escrito en Go 1.22/1.26, ejecutado mediante Systemd (`smart-backend.service`) en GCP VM dedicada, conectado a Google Cloud SQL (PostgreSQL).
* **Frontend:** Aplicación web moderna en Next.js 16 (TypeScript / Tailwind v4), administrada con PM2 (`smart-frontend`), desplegada en GCP VM dedicada.
* **Seguridad y SSL:** Ambos subdominios cuentan con certificados SSL válidos y activos expedidos por Let's Encrypt (vía Certbot + Nginx), corriendo sobre HTTPS/HTTP2.
* **ENTREGA 1 DESPLEGADA (Septiembre 2026):**
  1. **Matriz GRID de Programación (`/programacion`):** Reemplazo de vista simple por tabla GRID por colaborador con turnos `TD`, `TN`, `EX` y modal de **Programación Masiva de Turnos** con antiduplicados.
  2. **Bitácora de Incidentes (`/incidentes`):** Formulario con área **Drag & Drop para evidencia fotográfica**, filtro por estado y mapa de ubicación.
  3. **Visitas & Notificaciones (`/visitas`):** Flexibilización de validación de cédula/pasaporte y modal de **Notificación Asistida vía WhatsApp (`wa.me`) y Push Web**.
* **ENTREGA 2 DESPLEGADA (Noviembre 2026):**
  1. **Hermes Sentinel IA (`/hermes`):** Agente controlador y administrador autónomo en producción 24/7. Supervisa inactividad GPS en guardias con turno activo, escalamiento de alertas críticas SOS, rondas perimetrales incompletas y control de tiempos en flota pesada 360.
  2. **Backend & DB:** Tablas `hermes_config` y `hermes_alertas` en Cloud SQL, cuenta de servicio `hermes.agent@ncs365.com` (rol `SISADMIN`), endpoints `/api/v1/hermes/...` y worker en segundo plano en `smart-backend.service`.
  3. **Motor Conversacional Multitemático & IA Híbrida (`internal/hermes/chat.go`):**
     * Integración con **Gemini 2.0 Flash / 1.5 Flash** para razonamiento avanzado cuando se provee API Key.
     * **Motor Analítico Autónomo (`HERMES_ANALYTICS`):** Opera 100% offline y sin costo de API, resolviendo 11 intenciones operativas reales (Saludos, Rondas, Flota 360, Garita/Visitas, Búsqueda de personal por nombre, GPS en vivo, Turnos, Incidentes, Alertas, Casas/Contratos y Diagnóstico Global).
     * Telemetría GPS corregida para filtrar estrictamente balizas en vivo de los últimos 15 minutos e integración de catálogo completo de personal.
  4. **Frontend Producción:** Desplegado en `https://app.ncsecurity.net/hermes` bajo PM2 con métricas de salud, filtros por severidad, resolución de alertas, ajuste de parámetros y 7 botones de consulta rápida en el chat.
  5. **Aplicación Móvil para Administradores y Supervisores (Flutter):**
     * Enrutamiento dinámico en login: guardias van a `HomeScreen`, directivos/supervisores van a `HermesAdminScreen`.
     * Monitoreo con sondeo de 15s, vibración háptica en emergencias críticas, resolución de alertas desde el teléfono, chat con Hermes y supervisión GPS en vivo.
     * Interoperabilidad fluida con "Modo Guarda" para cubrir puestos o escanear puntos QR.
     * Artefacto compilado: `smart_residential_guarda.apk` (30 MB).

---

## 📌 2. Infraestructura y Redes en Google Cloud Platform (GCP)

| Servicio / Componente | Servidor / Recurso | IP Estática (Reservada) | Puerto Interno | Dominio / Subdominio | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Frontend (Next.js)** | `smart-residential-frontend-vm` | **`136.119.130.113`** | `3000` (PM2) | `https://app.ncsecurity.net` | 🟢 **ACTIVO (HTTPS/200 OK)** |
| **Backend (Go API)** | `smart-residential-backend-vm` | **`34.31.99.52`** | `8080` (Go) | `https://api.ncsecurity.net` | 🟢 **ACTIVO (HTTPS/200 OK)** |
| **Base de Datos** | Cloud SQL `ncs360-db-prod` | `127.0.0.1` (Proxy) | `5432` (PostgreSQL) | N/A (Interno) | 🟢 **ACTIVO (Systemd Proxy)** |

* **Región GCP:** `us-central1-a`
* **Tipo de Instancias:** `e2-medium` (Ubuntu 24.04 LTS)
* **Service Account GCP:** `338601123867-compute@developer.gserviceaccount.com` (con roles `roles/cloudsql.client` y scope `cloud-platform`).

---

## 📂 3. Estructura de Repositorios y Rutas Locales / Remotas

### 3.1. Backend (`smart-residential-backend`)
* **Repositorio GitHub:** `https://github.com/Marktuay/smart-residential-backend.git`
* **Ruta Local:** `/Users/informatica/Documents/smart-residential-backend`
* **Ruta en Servidor VM:** `/home/informatica/smart-residential-backend`
* **Servicio Systemd del Backend:** `/etc/systemd/system/smart-backend.service`
* **Servicio Systemd del Cloud SQL Proxy:** `/etc/systemd/system/cloud-sql-proxy.service`
* **Configuración Nginx:** `/etc/nginx/sites-available/backend`

### 3.2. Frontend (`smart-residential-frontend`)
* **Repositorio GitHub:** `https://github.com/Marktuay/smart-residential-frontend.git`
* **Ruta Local:** `/Users/informatica/Documents/smart-residential-frontend`
* **Ruta en Servidor VM:** `/home/informatica/smart-residential-frontend`
* **Proceso PM2:** `smart-frontend` (Port 3000)
* **Configuración Nginx:** `/etc/nginx/sites-available/frontend`
* **Variables de Entorno Production:** `/home/informatica/smart-residential-frontend/.env.production` -> `NEXT_PUBLIC_API_URL=https://api.ncsecurity.net`

---

## 🗄️ 4. Base de Datos & Migraciones

* **Motor:** PostgreSQL (Google Cloud SQL `skilful-gantry-345115:us-central1:ncs360-db-prod`)
* **Base de Datos:** `postgres` | **Usuario:** `postgres` | **Password:** `Mark8721!@#$`

### 🛠️ Migraciones Aplicadas en `incidentes`:
Para corregir el error del Centro de Mando (`pq: column "nivel_gravedad" does not exist`), se ejecutó la migración que añadió las siguientes columnas a la tabla `incidentes`:
* `reportado_por VARCHAR(100) DEFAULT 'SISTEMA'`
* `tipo VARCHAR(50) DEFAULT 'GENERAL'`
* `nivel_gravedad VARCHAR(20) DEFAULT 'MEDIA'`
* `evidencia_url TEXT`
* `fecha_creacion TIMESTAMP DEFAULT NOW()`
* `fecha_resolucion TIMESTAMP`

### 🔑 Credenciales Administrador Creadas para Pruebas:
* **Email:** `informatica@newcenturyni.com`
* **Password:** `Mark8721!@`
* **Rol:** `SISADMIN`
* **Residencial ID:** `Residencial-Las-Torres`

---

## ⚙️ 5. Puntos Clave Resueltos y Soluciones Técnicas

1. **Permisos de Cloud SQL Proxy:**
   * Se asignó la función `roles/cloudsql.client` a la cuenta de servicio predeterminada de Compute Engine y se activó el scope `https://www.googleapis.com/auth/cloud-platform`.

2. **Resolución DNS & SSL:**
   * La zona DNS de `ncsecurity.net` es administrada en **Cloudflare** (`jessica.ns.cloudflare.com` / `joaquin.ns.cloudflare.com`).
   * Se crearon los registros tipo A:
     * `app` -> `136.119.130.113`
     * `api` -> `34.31.99.52`
   * Certbot emitió los certificados SSL SSL/TLS válidos para ambos subdominios y configuró la renovación automática (`certbot.timer`).

3. **CORS & `src/lib/api.ts` (Frontend):**
   * Se corrigió la URL base en el cliente de axios en `src/lib/api.ts` para que tome dinámicamente `process.env.NEXT_PUBLIC_API_URL` o haga fallback a `https://api.ncsecurity.net/api/v1` (eliminando la referencia previa a `localhost:8080`).

4. **Centro de Mando & Autenticación (JWT):**
   * Se ajustó `src/app/(dashboard)/page.tsx` para validar la existencia del token JWT en `localStorage` antes de invocar a `/dashboard/stats`. En caso de respuesta HTTP 401 (sin autenticar / token expirado), se redirige automáticamente al usuario a `https://app.ncsecurity.net/login`.

---

## 🔒 6. Recomendaciones de Seguridad Continuas (Hardening)

1. **Variables de Entorno Sensibles:** Mantener el `JWT_SECRET` e información de conexión a DB fuera del código público de git.
2. **Encabezados Nginx:** Mantener activos los encabezados `X-Frame-Options`, `X-Content-Type-Options` y `Referrer-Policy`.
3. **CORS Restringido:** Limitar en el backend los orígenes permitidos explícitamente a `https://app.ncsecurity.net`.
