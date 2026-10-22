-- ==============================================================================
-- Migration: 000005_precio_venta_insumo.up.sql
-- ==============================================================================

-- Agrega columna de precio de venta base en USD por insumo (default 0.0)
-- Usado por la tabla de insumos v2 (Spec 004) para cálculo de margen y doble moneda USD/Bs.
-- La columna ahora se crea directamente en 000001_init_schema.up.sql para evitar duplicados.
-- ALTER TABLE insumos ADD COLUMN precio_venta_usd REAL DEFAULT 0.0;

-- Índice para consultas frecuentes de precio de venta
CREATE INDEX IF NOT EXISTS idx_insumos_precio_venta_usd ON insumos(precio_venta_usd);
