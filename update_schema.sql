CREATE TABLE IF NOT EXISTS residenciales (
    id VARCHAR(255) PRIMARY KEY,
    nombre VARCHAR(255) NOT NULL,
    direccion TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE incidentes RENAME COLUMN proyecto_id TO residencial_id;
ALTER TABLE logs_auditoria_sistema RENAME COLUMN project_id TO residencial_id;
