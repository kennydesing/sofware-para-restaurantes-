-- ==============================================================================
-- Migration: 000005_precio_venta_insumo.down.sql
-- ==============================================================================

-- Revertir la adición de la columna precio_venta_usd
-- SQLite ≥ 3.35 soporta DROP COLUMN directamente:
ALTER TABLE insumos DROP COLUMN precio_venta_usd;
