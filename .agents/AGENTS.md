# Memoria del Agente: Smart Residential Backend

Este archivo debe ser utilizado por el agente como contexto principal antes de sugerir o realizar modificaciones en el código. Define la arquitectura, reglas de negocio y estrategias de escalamiento del proyecto.

## 1. Arquitectura Base
- **Lenguaje:** Go 1.22+ (Utilizando el enrutador nativo `net/http` para endpoints como `GET /api/v1/incidentes`).
- **Base de Datos:** PostgreSQL en Google Cloud SQL, accedido vía el driver `github.com/lib/pq` y `database/sql` nativo (sin ORMs pesados).
- **Conexiones a BD:** El pool de conexiones está configurado con límites de concurrencia eficientes (`MaxOpenConns: 25`, `MaxIdleConns: 25`, `ConnMaxLifetime: 5 min`) en `internal/db/postgres.go`.

## 2. Paradigma Multi-Tenant (Inquilinos Múltiples)
- **Estrategia:** Aislamiento a nivel de fila (Row-Level Isolation / Shared Database, Shared Schema). Todos los residenciales conviven en la misma base de datos.
- **Regla Estricta:** TODAS las consultas SQL y escrituras (`INSERT`, `UPDATE`, `DELETE`, `SELECT`) DEBEN incluir invariablemente el filtro `residencial_id = $1`. Jamás se debe omitir.
- **Inyección del Inquilino:** El identificador del residencial entra por el header HTTP `X-Residencial-ID` y el `AuthMiddleware` se encarga de aislarlo e inyectarlo en el contexto (`r.Context().Value(middleware.ResidencialIDKey)`).

## 3. Seguridad y Autorización (RBAC)
- **Roles del Sistema:**
  - `SISADMIN`: Administrador del sistema global (Super-usuario).
  - `ADMIN`: Administrador de un residencial específico.
  - `RESIDENTE`: Habitante de un residencial.
  - `GUARDIA`: Personal de seguridad de un residencial.
- **Middleware RBAC:** La validación se hace a través de `middleware.RBACMiddleware`. 
- **Excepciones:** El rol `SISADMIN` tiene un bypass inyectado; se le permite el acceso universal a cualquier endpoint protegido por el middleware.

## 4. Auditoría Asíncrona (Eventos)
- **Trazabilidad:** Toda acción de lectura y escritura sensible (ej: `CONSULTA_INCIDENTES`, `CREA_INCIDENTE`) se guarda en la tabla `logs_auditoria_sistema`.
- **Rendimiento:** El registro de auditoría se debe disparar **siempre** de manera asíncrona usando goroutines (ej. `go registrarAuditoriaAsync(...)`) para no bloquear ni enlentecer la respuesta HTTP al cliente.
- **IP del Cliente:** Se debe usar la función helper `middleware.GetClientIP(r)` para extraer la IP real de los headers (`CF-Connecting-IP`, `X-Forwarded-For`), previniendo enmascaramientos por proxys inversos (Cloudflare/Load Balancers).

## 5. Escalabilidad y Extensibilidad Futura
- **Estructura del Proyecto:**
  - `cmd/api/`: Capa de presentación (servidor HTTP, endpoints).
  - `internal/model/`: Definición de estructuras y enums del dominio de negocio.
  - `internal/middleware/`: Interceptores de red (Auth, RBAC, etc.).
  - `internal/db/`: Configuración y gestores de estado hacia orígenes de datos.
- **Desarrollo a futuro:** Si las lógicas de negocio (`handlers`) crecen demasiado en `main.go`, deberán desacoplarse hacia capas de servicio dentro de `internal/service/` o `internal/repository/`.

## 6. Frontend y Configuración de CORS
- **Stack del Frontend:** La aplicación cliente se desarrollará utilizando **Next.js con Turbopack**.
- **Regla CORS:** Durante el desarrollo, el backend en Go permite solicitudes de todos los orígenes (`Access-Control-Allow-Origin: *`) en `internal/middleware/cors.go` para facilitar la conexión desde `localhost:3000`. Al pasar a producción, el middleware de CORS debe actualizarse estrictamente para apuntar al dominio final del frontend (ej: Vercel).
