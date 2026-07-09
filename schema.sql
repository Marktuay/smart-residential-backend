CREATE TABLE IF NOT EXISTS incidentes (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    titulo VARCHAR(100) NOT NULL,
    descripcion TEXT NOT NULL,
    prioridad VARCHAR(20) NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'PENDIENTE',
    categoria_id INT,
    creado_en TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidentes_residencial ON incidentes(residencial_id);

CREATE TABLE IF NOT EXISTS logs_auditoria_sistema (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    accion VARCHAR(100) NOT NULL,
    actor VARCHAR(100) NOT NULL,
    ip VARCHAR(50),
    detalles TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_logs_residencial ON logs_auditoria_sistema(residencial_id);

CREATE TABLE IF NOT EXISTS usuarios (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    rol VARCHAR(50) NOT NULL,
    creado_en TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usuarios_residencial ON usuarios(residencial_id);
