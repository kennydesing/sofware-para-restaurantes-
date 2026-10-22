-- ==============================================================================
-- Migración: 000006_empaques_cervezas_bebidas.up.sql
-- Propósito: Agregar metadatos de empaques predefinidos y personalizados
--            para cervezas y bebidas, garantizando conversiones exactas a unidad 'u'.
-- ==============================================================================

-- Las columnas de empaque ahora se crean directamente en 000001_init_schema.up.sql
-- para evitar duplicados al reiniciar la base de datos en desarrollo/pruebas.
-- ALTER TABLE insumos ADD COLUMN empaque_tipo TEXT NOT NULL DEFAULT 'UNITARIO'
--     CHECK (empaque_tipo IN ('UNITARIO', 'GAVERA_36', 'CAJA_24_BOT', 'CAJA_24_LATA', 'SIX_PACK_6', 'CAJA_12_ART', 'PERSONALIZADO'));
-- ALTER TABLE insumos ADD COLUMN empaque_nombre TEXT NOT NULL DEFAULT '';
-- ALTER TABLE insumos ADD COLUMN unidades_por_empaque INTEGER NOT NULL DEFAULT 1
--     CHECK (unidades_por_empaque >= 1);
-- ALTER TABLE insumos ADD COLUMN contenido_ml_por_unidad REAL NOT NULL DEFAULT 0.0
--     CHECK (contenido_ml_por_unidad >= 0.0);
-- ALTER TABLE insumos ADD COLUMN es_retornable INTEGER NOT NULL DEFAULT 0
--     CHECK (es_retornable IN (0, 1));

-- Índices de optimización para filtros rápidos en inventario
CREATE INDEX IF NOT EXISTS idx_insumos_empaque_tipo ON insumos(empaque_tipo);
CREATE INDEX IF NOT EXISTS idx_insumos_retornable ON insumos(es_retornable);
