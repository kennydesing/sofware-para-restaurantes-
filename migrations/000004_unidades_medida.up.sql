-- ==============================================================================
-- Migration: 000004_unidades_medida.up.sql
-- ==============================================================================

-- 1. CATÁLOGO DE UNIDADES DE MEDIDA
CREATE TABLE IF NOT EXISTS unidades_medida (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre      TEXT    NOT NULL,
    abreviatura TEXT    NOT NULL UNIQUE,
    categoria   TEXT    NOT NULL CHECK (categoria IN ('VOLUMEN','PESO','CONTEO','CULINARIA')),
    es_base     INTEGER NOT NULL DEFAULT 0 CHECK (es_base IN (0, 1)),
    factor_n    INTEGER NOT NULL DEFAULT 1 CHECK (factor_n > 0),
    factor_d    INTEGER NOT NULL DEFAULT 1 CHECK (factor_d > 0),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unidades_abrev    ON unidades_medida(abreviatura);
CREATE INDEX        IF NOT EXISTS idx_unidades_cat      ON unidades_medida(categoria);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unidades_base_cat ON unidades_medida(categoria) WHERE es_base = 1;

-- 2. FACTORES DE CONVERSIÓN EXPLÍCITOS
CREATE TABLE IF NOT EXISTS factores_conversion (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    unidad_origen_id  INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE CASCADE,
    unidad_destino_id INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE CASCADE,
    factor_n          INTEGER NOT NULL CHECK (factor_n > 0),
    factor_d          INTEGER NOT NULL CHECK (factor_d > 0),
    CONSTRAINT uq_conversion UNIQUE (unidad_origen_id, unidad_destino_id)
);

CREATE INDEX IF NOT EXISTS idx_conv_par ON factores_conversion(unidad_origen_id, unidad_destino_id);

-- 3. UNIDADES DE COMPRA POR INSUMO
CREATE TABLE IF NOT EXISTS insumo_unidades_compra (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    insumo_id    INTEGER NOT NULL REFERENCES insumos(id) ON DELETE CASCADE,
    unidad_id    INTEGER NOT NULL REFERENCES unidades_medida(id) ON DELETE RESTRICT,
    factor_n     INTEGER NOT NULL CHECK (factor_n > 0),
    factor_d     INTEGER NOT NULL CHECK (factor_d > 0),
    es_principal INTEGER NOT NULL DEFAULT 0 CHECK (es_principal IN (0, 1)),
    descripcion  TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_insumo_unidad UNIQUE (insumo_id, unidad_id)
);

CREATE INDEX        IF NOT EXISTS idx_iuc_insumo    ON insumo_unidades_compra(insumo_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_iuc_principal ON insumo_unidades_compra(insumo_id) WHERE es_principal = 1;

-- 4. EXTENDER recetas_escandallo
-- La columna unidad_receta_id se añade en 000001_init_schema.up.sql a partir de ahora.
-- Esta sentencia queda deshabilitada para evitar 'duplicate column name' en BD existentes:
-- ALTER TABLE recetas_escandallo ADD COLUMN unidad_receta_id INTEGER
--     REFERENCES unidades_medida(id) ON DELETE RESTRICT;

-- 5. SEED: CATÁLOGO ESTÁNDAR
INSERT OR IGNORE INTO unidades_medida (nombre, abreviatura, categoria, es_base, factor_n, factor_d) VALUES
    ('Mililitro',       'ml',     'VOLUMEN', 1, 1,          1),
    ('Litro',           'l',      'VOLUMEN', 0, 1000,       1),
    ('Galón US',        'gal',    'VOLUMEN', 0, 3785411784, 1000000),
    ('Onza líquida US', 'fl_oz',  'VOLUMEN', 0, 295735296,  10000000),
    ('Taza US',         'cup',    'VOLUMEN', 0, 236588237,  1000000),
    ('Cucharada',       'tbsp',   'VOLUMEN', 0, 147867648,  10000000),
    ('Cucharadita',     'tsp',    'VOLUMEN', 0, 492892159,  100000000),
    ('Gramo',           'g',      'PESO',    1, 1,          1),
    ('Kilogramo',       'kg',     'PESO',    0, 1000,       1),
    ('Miligramo',       'mg',     'PESO',    0, 1,          1000),
    ('Libra',           'lb',     'PESO',    0, 45359237,   100000),
    ('Onza',            'oz',     'PESO',    0, 283495231,  10000000),
    ('Unidad / Pieza',  'u',      'CONTEO',  1, 1,          1),
    ('Docena',          'dz',     'CONTEO',  0, 12,         1),
    ('Caja (genérica)', 'caja',   'CONTEO',  0, 1,          1),
    ('Bulto / Saco',    'bulto',  'CONTEO',  0, 1,          1),
    ('Atado / Manojo',  'atado',  'CONTEO',  0, 1,          1),
    ('Porción / Ración','porcion','CULINARIA',0, 1,          1);
