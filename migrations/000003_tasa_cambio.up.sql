-- Migración 000003: Tablas para gestión de tasas de cambio (BCV)
CREATE TABLE IF NOT EXISTS tasas_cambio (
    moneda TEXT PRIMARY KEY CHECK (moneda IN ('USD', 'EUR')),
    tasa REAL NOT NULL CHECK (tasa > 0),
    fuente TEXT NOT NULL DEFAULT 'BCV_AUTO', -- 'BCV_AUTO', 'MANUAL'
    fecha_actualizacion DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Valores iniciales de contingencia (Tasa oficial de arranque)
INSERT OR IGNORE INTO tasas_cambio (moneda, tasa, fuente, fecha_actualizacion) 
VALUES 
    ('USD', 877.90, 'BCV_AUTO', CURRENT_TIMESTAMP),
    ('EUR', 982.75, 'BCV_AUTO', CURRENT_TIMESTAMP);
