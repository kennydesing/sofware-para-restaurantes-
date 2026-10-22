-- ==============================================================================
-- Migración: 000006_empaques_cervezas_bebidas.down.sql
-- ==============================================================================

DROP INDEX IF EXISTS idx_insumos_retornable;
DROP INDEX IF EXISTS idx_insumos_empaque_tipo;

ALTER TABLE insumos DROP COLUMN es_retornable;
ALTER TABLE insumos DROP COLUMN contenido_ml_por_unidad;
ALTER TABLE insumos DROP COLUMN unidades_por_empaque;
ALTER TABLE insumos DROP COLUMN empaque_nombre;
ALTER TABLE insumos DROP COLUMN empaque_tipo;
