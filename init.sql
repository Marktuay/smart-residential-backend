CREATE TABLE IF NOT EXISTS incidentes (
    id SERIAL PRIMARY KEY,
    proyecto_id VARCHAR(255) NOT NULL,
    titulo VARCHAR(255) NOT NULL,
    descripcion TEXT NOT NULL,
    prioridad VARCHAR(50) NOT NULL,
    estado VARCHAR(50) NOT NULL,
    categoria_id INT,
    creado_en TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS logs_auditoria_sistema (
    id SERIAL PRIMARY KEY,
    project_id VARCHAR(255) NOT NULL,
    accion VARCHAR(255) NOT NULL,
    actor VARCHAR(255) NOT NULL,
    ip VARCHAR(255) NOT NULL,
    detalles TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
