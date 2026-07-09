CREATE TABLE IF NOT EXISTS puntos_qr (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    codigo_qr VARCHAR(255) UNIQUE NOT NULL,
    latitud DECIMAL(10,8),
    longitud DECIMAL(10,8)
);

CREATE TABLE IF NOT EXISTS programas_rondas (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    hora_inicio TIME NOT NULL,
    hora_fin TIME NOT NULL,
    recurrencia VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS programas_puntos (
    programa_id INT REFERENCES programas_rondas(id) ON DELETE CASCADE,
    punto_qr_id INT REFERENCES puntos_qr(id) ON DELETE CASCADE,
    orden INT NOT NULL,
    PRIMARY KEY(programa_id, punto_qr_id)
);

CREATE TABLE IF NOT EXISTS asignaciones_rondas (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    programa_id INT REFERENCES programas_rondas(id) ON DELETE CASCADE,
    usuario_id INT REFERENCES usuarios(id) ON DELETE SET NULL,
    fecha DATE NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'POR_ASIGNAR'
);

CREATE TABLE IF NOT EXISTS rondas_ejecutadas (
    id SERIAL PRIMARY KEY,
    asignacion_id INT REFERENCES asignaciones_rondas(id) ON DELETE CASCADE,
    punto_qr_id INT REFERENCES puntos_qr(id) ON DELETE CASCADE,
    fecha_verificacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    latitud DECIMAL(10,8),
    longitud DECIMAL(10,8)
);

CREATE TABLE IF NOT EXISTS rastreo_ubicacion (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    usuario_id INT REFERENCES usuarios(id) ON DELETE CASCADE,
    latitud DECIMAL(10,8) NOT NULL,
    longitud DECIMAL(10,8) NOT NULL,
    actualizado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS visitas (
    id SERIAL PRIMARY KEY,
    residencial_id VARCHAR(50) NOT NULL,
    nombre_visitante VARCHAR(150) NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'PROGRAMADA',
    fecha_esperada DATE NOT NULL,
    fecha_ingreso TIMESTAMP
);
