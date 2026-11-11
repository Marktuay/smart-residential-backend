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
- **Regla CORS:** Durante el desarrollo, el backend en Go permite solicitudes de todos los orígenes (`Access-Control-Allow-Origin: *`) en `internal/middleware/cors.go` para facilitar la conexión desde `localhost:3000`. Además, el middleware permite el encabezado custom `X-Residencial-ID` para que el frontend inyecte el tenant activo en cada petición HTTP (Preflight CORS). Al pasar a producción, el middleware de CORS debe actualizarse estrictamente para apuntar al dominio final del frontend (ej: Vercel).

## 7. Frontend, Identidad Visual y Seguridad
- **Integración de Autenticación E2E:** El inicio de sesión devuelve un JWT (`token`) y establece el identificador del inquilino (`X-Residencial-ID`), guardándose de forma segura en `localStorage` o cookies e inyectándose automáticamente en las solicitudes protegidas usando Axios Interceptors.
- **Identidad Corporativa (NCS365):** La interfaz adopta la identidad de "New Century Security S.A." usando componentes UI estilizados en tonos azules oscuros (`#1e293b`), iconos orientados a la seguridad (Lucide-react: `Shield`, `Siren`, etc.) en un tono amarillo dorado corporativo (`#FACC15`) y logotipos adaptables a fondos oscuros.
- **Gestión de Layouts (Next.js):** Se implementó un diseño de layouts anidados que aísla la cabecera y el panel lateral dentro de `(dashboard)` para no interferir con las pantallas de inicio de sesión (`/login`), previniendo choques de estilos CSS globales y mejorando la carga progresiva.

## 8. Finanzas y Gestión de Cobros
- **Integración Externa:** Todo lo relacionado con finanzas y gestión de cobros se manejará a través de una API externa. 
- **Sistema Existente:** Se consumirán los datos de un sistema de gestión de cobros previamente construido, el cual ya está desarrollado utilizando el stack **Next.js, Nest.js y PostgreSQL**. El backend en Go de este proyecto actuará únicamente como consumidor/cliente de esa API para la información financiera, en lugar de duplicar la lógica de cobro o la base de datos de finanzas localmente.

## 9. Rondas, Puntos QR y Control de Accesos
- **Estilos en Línea (React):** Para evitar problemas de configuración o dependencias rotas con Tailwind CSS, las pantallas de Gestión de Rondas, Visitas y Supervisores han sido reescritas usando estilos en línea puros en React (`style={{ ... }}`). Se mantiene estrictamente la paleta de colores de la identidad **NCS365**.
- **Gestión de Supervisores:** Se agregó la pantalla de Supervisores (`/supervisores`) al menú lateral de Operaciones, para tener control sobre el personal.
- **Relación QR - Casas:** Los `puntos_qr` de las rondas de seguridad ahora pueden estar asociados opcionalmente a una `Casa`. La tabla `casas` posee el flag `tiene_contrato_seguridad`. En el frontend, las pantallas de `/rondas` y `/puntos-qr` muestran detalles sobre el propietario y un icono distintivo de escudo (verde para contrato activo, rojo para inactivo) que ayuda al guardia durante su recorrido. Se integró el formulario de asociación al crear puntos de control en ambas vistas.

## 10. Jerarquía de Residenciales y Mapas
- **Pestaña y Gestión de Residenciales:** Ubicada de forma prominente como primera pestaña en `/residencial`. Permite visualizar todas las urbanizaciones/condominios registrados en tarjetas corporativas con indicadores de estado (`ACTIVO`), contactos y parámetros operativos.
- **Acceso RBAC:** Los roles `ADMIN` y `SISADMIN` tienen autorización para listar (`GET /api/v1/residenciales`), crear nuevas instalaciones (`POST /api/v1/residenciales`) y configurar sus parámetros (`PUT /api/v1/residencial/config`).
- **Barra de Residencial Activo:** Implementada en la cabecera superior de `/residencial` con selector desplegable sincronizado con `localStorage.getItem('residencial_id')`, métricas rápidas (casas y habitantes asociados) y botón directo `+ Nuevo Residencial`.
- **Configuración Geográfica y Satelital:** Cada residencial almacena Latitud Centro, Longitud Centro y Zoom por Defecto (10–18). Estos valores alimentan el encuadre inicial en los mapas operativos de `/map` y control de rondas.
- **Jerarquía:** Antes de registrar Ubicaciones, Casas o Puntos QR, se debe registrar el Residencial base. En el modal de registro de viviendas se muestra explícitamente el residencial asignado.

## 11. Programación y Gestión de Turnos
- **Aislamiento por Residencial:** Planificación de turnos de guardias y supervisores realizada de forma aislada por residencial activo (`X-Residencial-ID`), con acceso restringido exclusivamente a los roles `ADMIN` y `SISADMIN`.
- **Arquitectura y Endpoints:** Expuestos los servicios REST bajo `/api/v1/turnos` en el backend en Go, con inserción asíncrona de auditorías (`CREAR_TURNO`, `ACTUALIZAR_TURNO`, `ELIMINAR_TURNO`).
- **Interfaz (Calendario):** Implementada una cuadrícula de calendario mensual interactiva en `/programacion` con colores corporativos **NCS365** y modales interactivos para el flujo completo CRUD (Crear, Editar, Consultar y Cancelar).

## 12. Control de Flota Pesada, Maquinaria Amarilla y Checklist 360
- **Aislamiento y Tablas:** Tablas en PostgreSQL (`flota_vehiculos`, `solicitudes_flota`, `checklist_360_inspecciones`, `checklist_items_detalles`) con aislamiento estricto por `residencial_id = $1`.
- **Roles RBAC del Módulo:** `OPERADOR_OPERACIONES` (apertura de solicitud y ruta), `GESTOR_TRANSPORTE` (asignación de unidad/chofer), `MECANICO_TALLER` (inspección 360 mecánica y horómetro/km), `GUARDIA_GARITA` (EPP de conductor y control de salida/llegada), `AUDITOR_FLOTA` (trazabilidad global), con bypass para `SISADMIN`.
- **Máquina de Estados Finita (FSM):** Ciclo irreversible de 10 estados (`DRAFT` ➔ `ASIGNADA` ➔ `INSPECCION_TALLER_PENDIENTE` ➔ `APROBADA_TALLER` / `RECHAZADA_TALLER` ➔ `GARITA_SALIDA_PENDIENTE` ➔ `EN_TRANSITO` ➔ `GARITA_LLEGADA_PENDIENTE` ➔ `CHECKIN_CONFORME` / `CHECKIN_CON_OBSERVACION` ➔ `INSPECCION_CIERRE_TALLER` ➔ `CERRADO_FINALIZADO`).
- **PWA e Interfaz de Usuario:** Formulario optimizado para tablets IP68 en `/flota/inspeccion` con captura fotográfica, dashboard en `/flota`, catálogo en `/flota/vehiculos`, y contenedor enmarcado con borde dorado (`#FACC15`) y badge `MÓDULO 360` en el `Sidebar.tsx`.

## 13. Rastreo Satelital GPS en Vivo e Histórico (Tracking)
- **Endpoints:**
  - `POST /api/v1/tracking/location`: Baliza periódica de posicionamiento transmitida por el personal operativo.
  - `DELETE /api/v1/tracking/location`: Remueve la baliza activa al apagar patrullaje o cerrar sesión.
  - `GET /api/v1/tracking/ubicaciones`: Recupera guardias activos transmitiendo en tiempo real (ventana extendida a 15 minutos).
  - `GET /api/v1/tracking/historial`: Consulta trazado de rutas históricas por `usuario_id`, `fecha_inicio` y `fecha_fin`.
- **Normalización y Conversión de Zonas Horarias:** La base de datos almacena marcas temporales en UTC (`registrado_en`). El backend incorpora el parser inteligente `ParseFechaFiltro` en `internal/repository/tracking.go`, capaz de procesar cadenas ISO-8601 UTC (`Z`) o interpretar horas locales (asumiendo Nicaragua `America/Managua` UTC-6) y convertirlas a UTC de forma transparente, previniendo discrepancias y garantizando el trazado de polilíneas exactas en el mapa web.
- **Segmentación Inteligente en Mapa Web:** En `src/components/map/MapContainerComponent.tsx`, el mapa segmenta automáticamente la trayectoria si la diferencia entre dos puntos excede 2.5 minutos o existe un salto espacial > 200m sin puntos intermedios. Previene el trazado de líneas rectas irreales sobre casas o muros, dibujando segmentos limpios y líneas de advertencia para brechas de señal.
- **Servicio en Primer Plano en App Móvil:** La app Android (`smart_residential_mobile`) implementa `AndroidSettings` con `ForegroundNotificationConfig` (`enableWakeLock: true`, `distanceFilter: 8m`), impidiendo que el Doze Mode de Android congele las transmisiones GPS en el bolsillo o con la pantalla apagada.
- **Aislamiento Multi-Tenant & Modo Global:** Los usuarios con rol `SISADMIN` o con residencial `GLOBAL` pueden consultar ubicaciones e historial de forma consolidada para todas las instalaciones, mientras que los roles `ADMIN` visualizan estrictamente el personal de su residencial.

## 14. Aplicación Móvil para Guardias y Patrulleros (Flutter APK)
- **Ubicación del Proyecto y Compilado:** Desarrollada en Flutter (Dart 3.2.6) en `/Users/informatica/Documents/smart_residential_mobile`. Artefacto release para distribución: `smart_residential_guarda.apk` (30 MB).
- **Módulos Operativos Integrados:**
  - **Patrullaje GPS Continuo en Segundo Plano:** Servicio de baliza que transmite coordenadas cada 15 segundos incluso con la pantalla apagada o aplicación minimizada (`_stopGpsOnBackground = false`), con contador visual de paquetes y confirmación de recepción.
  - **Escáner QR para Rondas Perimetrales:** Captura mediante cámara en tiempo real (`mobile_scanner`), control de linterna y registro de marcaciones georreferenciadas hacia `POST /api/v1/rondas/marcacion`.
  - **Control de Garita y Visitas:** Bitácora en vivo con búsqueda por nombre, placa o número de casa, registro de nuevas visitas y transiciones rápidas de estado (`EN_CURSO` y `FINALIZADA`).
  - **Despacho de Incidentes de Campo:** Formulario con categorización, chips de severidad (`BAJA`, `MEDIA`, `ALTA`, `CRITICA`), coordenadas GPS auto-detectadas y envío a `POST /api/v1/incidentes`.
  - **Gestión de Turnos y Asistencia:** Visualización de turnos programados y registro de check-in (`POST /api/v1/turnos/{id}/asistencia`).
  - **Botón de Emergencia SOS:** Disparo de alerta crítica inmediata en garita y central de monitoreo con un toque.

## 15. Agente Autónomo Controlador y Administrador en Producción (Hermes Sentinel)
- **Identidad de Servicio y Rol:** Usuario de sistema `hermes.agent@ncs365.com` con rol `SISADMIN` y `residencial_id: GLOBAL`. Trazabilidad total de cada intervención en `logs_auditoria_sistema` (`actor = "HERMES_CONTROLLER"`).
- **Tablas en PostgreSQL:**
  - `hermes_config`: Parámetros de monitoreo (`intervalo_evaluacion_seg`, `max_inactividad_gps_mins`, banderas para alertas SOS, rondas y flota por residencial o `GLOBAL`).
  - `hermes_alertas`: Anomalías registradas (`INACTIVIDAD_GPS`, `BOTON_PANICO_SOS`, `RONDA_VENCIDA`, `INCIDENTE_CRITICO`, `FLOTA_ANOMALIA`), con severidades (`BAJA`, `MEDIA`, `ALTA`, `CRITICA`), deduplicación en ventana de 15 minutos y ciclo de vida (`PENDIENTE`, `NOTIFICADO`, `ATENDIDO`, `DESCARTADO`).
- **Motor Centinela en Go (`internal/hermes`):**
  - Corre en goroutines en segundo plano (intervalo por defecto: 60s) coordinado en `cmd/api/main.go`.
  - **Inactividad GPS:** Detecta guardias con turno activo (`PRESENTE` o en horario vigente) sin emitir balizas GPS por más de `max_inactividad_gps_mins` (defecto 5 min) y genera alerta `ALTA`.
  - **Incidentes Críticos y SOS:** Detecta alertas de pánico e incidentes de severidad `CRITICA` abiertos y genera escalamiento inmediato.
  - **Rondas Vencidas:** Identifica rondas perimetrales no completadas al expirar su horario con tolerancia.
  - **Flota 360:** Supervisa retenciones anormales de vehículos en taller o garita de salida.
- **Endpoints REST Expuestos:**
  - `GET /api/v1/hermes/status`: Estado, uptime, versión y métricas de alertas activas.
  - `GET /api/v1/hermes/alertas`: Listado filtrable de alertas por residencial, severidad y estado.
  - `PUT /api/v1/hermes/alertas/{id}/resolver`: Resolución o descarte de alertas por supervisores.
  - `GET /api/v1/hermes/config` & `PUT /api/v1/hermes/config`: Lectura y ajuste de parámetros de supervisión.
  - `POST /api/v1/hermes/evaluar`: Disparo manual de ciclo de evaluación.
  - `POST /api/v1/hermes/chat`: Asistente conversacional con IA híbrida y telemetría en tiempo real:
    - **Soporte Gemini:** Invoca `gemini-2.0-flash` con fallback a `gemini-1.5-flash` cuando se provee API Key (`hermes_config` o `GEMINI_API_KEY`).
    - **Motor Analítico Autónomo (`HERMES_ANALYTICS`):** Opera 100% offline y sin costo de API, resolviendo 11 intenciones operativas reales:
      1. *Saludos y Ayuda:* Saludo corporativo de NCS365 y menú de capacidades.
      2. *Rondas y Recorridos:* Programas activos, porcentaje de cumplimiento, rondas atrasadas y puntos QR marcados.
      3. *Flota Pesada 360:* Vehículos operando vs taller, solicitudes de movimiento y máquina de estados FSM.
      4. *Garita y Visitas:* Visitantes del día, vehículos activos en predio (`EN_CURSO`), placas y casas destino.
      5. *Búsqueda Individual de Personal:* Consulta por nombre, email o ID de cualquier colaborador (ej. "Test", "Mario"), retornando rol, estado en nómina, horario de turno hoy y telemetría satelital GPS.
      6. *Personal y Balizas GPS (< 15 min):* Telemetría en vivo de guardias activos, velocidad, precisión y coordenadas.
      7. *Programación y Turnos:* Turnos asignados del día y asistencia confirmada (`PRESENTE`/`PENDIENTE`).
      8. *Incidentes y Botón SOS:* Novedades de las últimas 24 horas clasificadas por gravedad y estado.
      9. *Alertas Centinela:* Bandeja de anomalías detectadas automáticamente por Hermes.
      10. *Comunidad y Casas:* Censo inmobiliario y porcentaje de cobertura con contrato de seguridad activo.
      11. *Diagnóstico Operativo Global:* Tablero ejecutivo con semáforo de riesgo consolidado.

## 16. Experiencia Móvil de Administrador y Supervisor (Hermes en Flutter)
- **Enrutamiento por Rol en Login (`lib/main.dart`):**
  - Roles operativos (`GUARDIA`): Redirigen a la interfaz de campo tradicional de rondas, garita y baliza GPS (`HomeScreen`).
  - Roles de mando (`ADMIN`, `SISADMIN`, `SUPERVISOR`): Redirigen automáticamente a la consola móvil de mando de **Hermes Sentinel** (`HermesAdminScreen`).
- **Módulos Integrados en la App Móvil (`lib/screens/hermes_admin_screen.dart`):**
  - **Pestaña 1 (Bandeja de Alertas en Tiempo Real):** Monitoreo activo con polling de 15 segundos y vibración háptica fuerte ante alertas críticas. Filtros dinámicos por estado (`PENDIENTE`, `ATENDIDO`, `DESCARTADO`) y severidad (`CRITICA`, `ALTA`, etc.), con modal para resolución y registro de notas de campo directamente a la base de datos de producción.
  - **Pestaña 2 (Chat Operativo con Hermes IA):** Interacción directa con el agente controlador para consultar el estado de las instalaciones, rondas atrasadas, incidentes y disponibilidad del personal con respuestas inmediatas contextualizadas y 7 chips de consulta rápida.
  - **Pestaña 3 (Supervisión GPS en Vivo):** Listado del personal en servicio activo transmitiendo coordenadas en tiempo real, con velocidad y precisión en metros.
  - **Interoperabilidad:** Menú superior que permite a supervisores conmutar al "Modo Guarda" en caso de necesitar escanear puntos QR o cubrir un puesto sin cerrar sesión.
- **Artefacto Compilado:** Release para Android `smart_residential_guarda.apk` (30 MB) generado con `flutter build apk --release` y validado con `flutter analyze` (0 errores, 0 advertencias).





